package features_test

import (
	"fmt"
	"github.com/buildpack/libbuildpack/application"
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

func TestUnitNginx(t *testing.T) {
	spec.Run(t, "Nginx", testNginx, spec.Report(report.Terminal{}))
}

func testNginx(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("Nginx web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.NginxFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewNginxFeature(
				factory.Build.Application,
				config.BuildpackYAML{Config: config.Config{
					WebServer:    config.Nginx,
					WebDirectory: "some-dir",
				}},
			)
		})

		it("is detected when Nginx web server requested", func() {
			Expect(factory).NotTo(BeNil())
			Expect(p.IsNeeded()).To(BeTrue())
		})

		it("sets start command on the layers object", func() {
			layer := factory.Build.Layers.Layer("layer-1")
			Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

			nginxConfPath := filepath.Join(factory.Build.Application.Root, "nginx.conf")
			procsPath := filepath.Join(layer.Root, "procs.yml")

			Expect(nginxConfPath).To(BeARegularFile())
			Expect(procsPath).To(BeARegularFile())

			buf, err := ioutil.ReadFile(nginxConfPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(ContainSubstring(fmt.Sprintf("server unix:%s", layer.Root)))
			Expect(string(buf)).To(ContainSubstring("some-dir"))

			procs, err := procmgr.ReadProcs(procsPath)
			Expect(err).ToNot(HaveOccurred())


			Expect(procs.Processes).To(Equal(map[string]procmgr.Proc{
				"nginx": procmgr.Proc{
					Command: "nginx",
					Args:    []string{"-p", factory.Build.Application.Root, "-c", filepath.Join(factory.Build.Application.Root, "nginx.conf")},
				},
			}))
		})

		it("Nginx web server is not present", func() {
			p = features.NewNginxFeature(
				application.Application{},
				config.BuildpackYAML{Config: config.Config{
					WebServer:    "some-other-webserver",
					WebDirectory: "some-dir",
				}},
			)
			Expect(p.IsNeeded()).To(BeFalse())
		})

	})
}
