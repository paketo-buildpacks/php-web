package features_test

import (
	"fmt"
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"
	"github.com/cloudfoundry/php-web-cnb/procmgr"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestUnitPhpFpm(t *testing.T) {
	spec.Run(t, "PhpFpm", testPhpFpm, spec.Report(report.Terminal{}))
}

func testPhpFpm(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("PhpFpm web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.PhpFpmFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewPhpFpmFeature(
				factory.Build.Application,
				config.BuildpackYAML{Config: config.Config{
					WebServer:    config.Nginx,
					WebDirectory: "some-dir",
				}},
			)
		})

		it("is detected when PhpFpm web server requested", func() {
			Expect(factory).NotTo(BeNil())
			Expect(p.IsNeeded()).To(BeTrue())
		})


		for _, path := range []string{filepath.Join(".php.fpm.d", "user.conf"), ""} {
			it(fmt.Sprintf("sets start command on the layers object with path [%s]", path), func() {
				layer := factory.Build.Layers.Layer("layer-1")
				if path != "" {
					Expect(helper.WriteFile(filepath.Join(factory.Build.Application.Root, path), 0644, "")).To(Succeed())
				}

				Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

				phpfpmConfPath := filepath.Join(layer.Root, "etc", "php-fpm.conf")
				procsPath := filepath.Join(layer.Root, "procs.yml")

				Expect(phpfpmConfPath).To(BeARegularFile())
				Expect(procsPath).To(BeARegularFile())

				buf, err := ioutil.ReadFile(phpfpmConfPath)
				Expect(err).ToNot(HaveOccurred())

				// only add *.conf if user provided user.conf file exists
				if path != "" {
					Expect(string(buf)).To(ContainSubstring(fmt.Sprintf(`include=%s`, filepath.Join(factory.Build.Application.Root, ".php.fpm.d", "*.conf"))))
				}

				procs, err := procmgr.ReadProcs(procsPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(procs.Processes).To(Equal(map[string]procmgr.Proc{
					"php-fpm": procmgr.Proc{
						Command: "php-fpm",
						Args:    []string{"-p", layer.Root, "-y", filepath.Join(layer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(layer.Root, "etc")},
					},
				}))
			})
		}



		it("PhpFpm web server is not present", func() {
			p = features.NewPhpFpmFeature(
				application.Application{},
				config.BuildpackYAML{Config: config.Config{
					WebServer:    "some-other-webserver",
					WebDirectory: "some-dir",
				}},
			)
			Expect(p.IsNeeded()).To(BeFalse()) // Always true for now
		})

	})
}
