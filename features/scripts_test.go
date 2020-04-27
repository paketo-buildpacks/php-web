package features_test

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/paketo-buildpacks/php-web/config"
	"github.com/paketo-buildpacks/php-web/features"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestUnitScripts(t *testing.T) {
	spec.Run(t, "Scripts", testScripts, spec.Report(report.Terminal{}))
}

func testScripts(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("no web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.ScriptsFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewScriptsFeature(
				features.FeatureConfig{
					App:      factory.Build.Application,
					IsWebApp: false,
				},
			)
		})

		when("checking if IsNeeded", func() {
			it("because this is a PHP script", func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "some-dir", "index.php"), "")

				Expect(factory).NotTo(BeNil())
				Expect(p.IsNeeded()).To(BeTrue())
			})

			it("because this is a web app", func() {
				p = features.NewScriptsFeature(
					features.FeatureConfig{
						BpYAML:   config.BuildpackYAML{Config: config.Config{
							WebServer:   config.ApacheHttpd,
							WebDirectory: "some-dir",
						}},
						App:      factory.Build.Application,
						IsWebApp: true,
					},
				)
				Expect(p.IsNeeded()).To(BeFalse())
			})
		})

		when("starting a PHP script", func() {
			it("starts a script using default `app.php`", func() {
				for _, script := range config.DefaultCliScripts {
					layer := factory.Build.Layers.Layer(fmt.Sprintf("layer-%s", script))
					scriptName := filepath.Join(factory.Build.Application.Root, script)
					err := helper.WriteFile(scriptName, 0655, "")
					Expect(err).ToNot(HaveOccurred())

					p = features.NewScriptsFeature(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{Config: config.Config{
								WebServer:   config.ApacheHttpd,
								WebDirectory: "some-dir",
							}},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
					)
					Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

					command := fmt.Sprintf("php %s/%s", factory.Build.Application.Root, script)
					Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
						Processes: []layers.Process{
							{Type: "task", Command: command, Direct: false},
							{Type: "web", Command: command, Direct: false},
						},
					}))

					os.Remove(scriptName)
				}
			})

			it("starts a script using custom script path/name", func() {
				layer := factory.Build.Layers.Layer("layer-1")

				p = features.NewScriptsFeature(
					features.FeatureConfig{
						BpYAML:   config.BuildpackYAML{
							Config: config.Config{
							Script: "relative/path/to/my/script.php",
						}},
						App:      factory.Build.Application,
						IsWebApp: true,
					},
				)
				Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

				command := fmt.Sprintf("php %s/%s", factory.Build.Application.Root, "relative/path/to/my/script.php")
				Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
					Processes: []layers.Process{
						{Type: "task", Command: command, Direct: false},
						{Type: "web", Command: command, Direct: false},
					},
				}))
			})
		})

	})
}
