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
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("php web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.ProcMgrFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewProcMgrFeature(
				filepath.Join(factory.Build.Buildpack.Root, "procMgr"),
				config.BuildpackYAML{Config: config.Config{
					WebServer: config.ApacheHttpd,
				}},
			)
		})

		when("IsNeeded", func() {
			it("is detected when httpd requested", func() {
				Expect(factory).NotTo(BeNil())
				Expect(p.IsNeeded()).To(BeTrue())
			})

			it("is detected when nginx requested", func() {
				p = features.NewProcMgrFeature(
					"",
					config.BuildpackYAML{Config: config.Config{
						WebServer: config.Nginx,
					}},
				)

				Expect(factory).NotTo(BeNil())
				Expect(p.IsNeeded()).To(BeTrue())
			})

			it("is not detected for some other webserver", func() {
				p = features.NewProcMgrFeature(
					"",
					config.BuildpackYAML{Config: config.Config{
						WebServer: "other-webserver",
					}},
				)

				Expect(factory).NotTo(BeNil())
				Expect(p.IsNeeded()).To(BeFalse())
			})
		})
		when("EnableFeature", func() {
			it("installs itself and adds procs", func() {

				currentLayer := factory.Build.Layers.Layer("layer-1")

				Expect(helper.WriteFile(
					filepath.Join(factory.Build.Buildpack.Root, "procMgr"),
					os.ModePerm,
					"some content")).To(Succeed())
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
