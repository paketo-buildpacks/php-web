package features_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitPhpWebServer(t *testing.T) {
	spec.Run(t, "PhpWebServer", testPhpWebServer, spec.Report(report.Terminal{}))
}

func testPhpWebServer(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("php web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.PhpWebServerFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewPhpWebServerFeature(
				features.FeatureConfig{
					App: factory.Build.Application,
					BpYAML: config.BuildpackYAML{Config: config.Config{
						WebServer:    config.PhpWebServer,
						WebDirectory: "some-dir",
					}},
					IsWebApp: true,
				},
			)
		})

		when("checking if IsNeeded", func() {
			when("and we have a web app and php web has been requested", func() {
				it("is true", func() {
					test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "some-dir", "index.php"), "")

					Expect(factory).NotTo(BeNil())
					Expect(p.IsNeeded()).To(BeTrue())
				})
			})

			when("and php web has not been requested", func() {
				it("is false", func() {
					p = features.NewPhpWebServerFeature(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{Config: config.Config{
								WebServer:   "some-other-webserver",
								WebDirectory: "some-dir",
							}},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
					)
					Expect(p.IsNeeded()).To(BeFalse())
				})
			})

			when("and it is not a web app", func() {
				it("is false", func() {
					p = features.NewPhpWebServerFeature(
						features.FeatureConfig {
							BpYAML: config.BuildpackYAML{Config: config.Config{}},
							App: factory.Build.Application,
							IsWebApp: false,
						},
					)
					Expect(p.IsNeeded()).To(BeFalse())
				})
			})
		})

		it("sets start command on the layers object", func() {
			expectedCommand := fmt.Sprintf(
				"php -S 0.0.0.0:$PORT -t %s",
				filepath.Join(factory.Build.Application.Root, "some-dir"),
			)
			layer := factory.Build.Layers.Layer("layer-1")
			Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())
			Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
				Processes: []layers.Process{
					{Type: "task", Command: expectedCommand, Direct: false},
					{Type: "web", Command: expectedCommand, Direct: false},
				},
			}))
		})

	})
}
