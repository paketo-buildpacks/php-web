package features_test

import (
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/paketo-buildpacks/php-web/config"
	"github.com/paketo-buildpacks/php-web/features"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestUnitPhp(t *testing.T) {
	spec.Run(t, "Php", testPhp, spec.Report(report.Terminal{}))
}

func testPhp(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("init Php", func() {
		var (
			factory *test.BuildFactory
			p       features.PhpFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewPhpFeature(
				features.FeatureConfig{
					BpYAML:   config.BuildpackYAML{Config: config.Config{
						WebDirectory: "some-dir",
					}},
					App:      factory.Build.Application,
				},
			)
		})

		when("checking if IsNeeded", func() {
			when("and we have a php app", func() {
				it("is true", func() {
					test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "some-dir", "index.php"), "")

					Expect(factory).NotTo(BeNil())
					Expect(p.IsNeeded()).To(BeTrue())
				})
			})
		})

		it("sets env vars and writes php.ini file", func() {
			layer := factory.Build.Layers.Layer("layer-1")
			Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

			Expect(filepath.Join(layer.Root, "etc", "php.ini")).To(BeARegularFile())
			Expect(layer).To(test.HaveOverrideSharedEnvironment("PHPRC", filepath.Join(layer.Root, "etc")))
			Expect(layer).To(test.HaveOverrideSharedEnvironment("PHP_INI_SCAN_DIR", filepath.Join(factory.Build.Application.Root, ".php.ini.d")))

		})

	})
}
