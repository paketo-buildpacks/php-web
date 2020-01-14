package features_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitProcMgr(t *testing.T) {
	spec.Run(t, "ProcMgr", testProcMgr, spec.Report(report.Terminal{}))
}

func testProcMgr(t *testing.T, when spec.G, it spec.S) {
	var (
		factory *test.BuildFactory
		p       features.ProcMgrFeature
	)

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
	})

	when("ProcMgr is present", func() {
		when("some other web server is requested", func() {
			it("is false", func() {
				p = features.NewProcMgrFeature(
					features.FeatureConfig{
						BpYAML: config.BuildpackYAML{Config: config.Config{
							WebServer: "other-webserver",
						}},
					},
					"",
				)

				Expect(factory).NotTo(BeNil())
				Expect(p.IsNeeded()).To(BeFalse())
			})
		})

		for _, webServer := range []string{config.Nginx, config.ApacheHttpd} {
			when(fmt.Sprintf("using web server %s", webServer), func() {
				it.Before(func() {
					p = features.NewProcMgrFeature(
						features.FeatureConfig{
							BpYAML: config.BuildpackYAML{Config: config.Config{
								WebServer: webServer,
							}},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						filepath.Join(factory.Build.Buildpack.Root, "procMgr"),
					)
				})

				when("checking if IsNeeded", func() {
					when(fmt.Sprintf("and we have a web app and has webserver %s", webServer), func() {
						it("is true", func() {
							Expect(factory).NotTo(BeNil())
							Expect(p.IsNeeded()).To(BeTrue())
						})
					})

					when("and it is not a web app", func() {
						it("is false", func() {
							p = features.NewProcMgrFeature(
								features.FeatureConfig{
									BpYAML:   config.BuildpackYAML{Config: config.Config{}},
									App:      factory.Build.Application,
									IsWebApp: false,
								},
								"",
							)
							Expect(p.IsNeeded()).To(BeFalse())
						})
					})
				})
			})
		}

		when("EnableFeature", func() {
			it("installs itself and adds procs", func() {
				currentLayer := factory.Build.Layers.Layer("layer-1")
				procMgrPath := filepath.Join(factory.Build.Buildpack.Root, "procMgr")

				p = features.NewProcMgrFeature(
					features.FeatureConfig{
						BpYAML: config.BuildpackYAML{Config: config.Config{
							WebServer: config.ApacheHttpd,
						}},
						App:      factory.Build.Application,
						IsWebApp: true,
					},
					procMgrPath,
				)

				Expect(helper.WriteFile(procMgrPath, os.ModePerm,"some content")).To(Succeed())

				Expect(p.EnableFeature(factory.Build.Layers, currentLayer)).To(Succeed())

				Expect(filepath.Join(currentLayer.Root, "bin", "procmgr")).To(BeARegularFile())
				procsYMLPath := filepath.Join(currentLayer.Root, "procs.yml")

				Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
					Processes: []layers.Process{
						{Type: "web", Command: fmt.Sprintf("procmgr %s", procsYMLPath), Direct: false},
					},
				}))
			})
		})

	})
}
