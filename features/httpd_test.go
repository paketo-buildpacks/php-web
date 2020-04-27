package features_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/php-web/procmgr"

	"github.com/paketo-buildpacks/php-web/config"
	"github.com/paketo-buildpacks/php-web/features"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitHttpd(t *testing.T) {
	spec.Run(t, "Httpd", testHttpd, spec.Report(report.Terminal{}))
}

func testHttpd(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("Apache web server is present", func() {
		var (
			factory *test.BuildFactory
			p       features.HttpdFeature
		)

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			p = features.NewHttpdFeature(
				features.FeatureConfig{
					BpYAML: config.BuildpackYAML{Config: config.Config{
						WebServer:           config.ApacheHttpd,
						WebDirectory:        "some-dir",
						ServerAdmin:         "my-admin@example.com",
						EnableHTTPSRedirect: true,
					}},
					App:      factory.Build.Application,
					IsWebApp: true,
				},
			)
		})

		when("checking if IsNeeded", func() {
			when("and we have a web app and HTTPD has been requested", func() {
				it("is true", func() {
					test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "some-dir", "index.php"), "")

					Expect(factory).NotTo(BeNil())
					Expect(p.IsNeeded()).To(BeTrue())
				})
			})

			when("and Apache web server has not been requested", func() {
				it("is false", func() {
					p = features.NewHttpdFeature(
						features.FeatureConfig{
							BpYAML: config.BuildpackYAML{Config: config.Config{
								WebServer:    "some-other-webserver",
								WebDirectory: "some-dir",
								ServerAdmin:  "my-admin@example.com",
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
					p = features.NewHttpdFeature(
						features.FeatureConfig{
							BpYAML: config.BuildpackYAML{Config: config.Config{
								WebServer: config.ApacheHttpd,
							}},
							App:      factory.Build.Application,
							IsWebApp: false,
						},
					)
					Expect(p.IsNeeded()).To(BeFalse())
				})
			})
		})

		it("sets start command on the layers object", func() {
			layer := factory.Build.Layers.Layer("layer-1")
			Expect(p.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

			httpdConfPath := filepath.Join(factory.Build.Application.Root, "httpd.conf")
			procsPath := filepath.Join(layer.Root, "procs.yml")

			Expect(httpdConfPath).To(BeARegularFile())
			Expect(procsPath).To(BeARegularFile())

			buf, err := ioutil.ReadFile(httpdConfPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(ContainSubstring("127.0.0.1:9000"))
			Expect(string(buf)).To(ContainSubstring("some-dir"))
			Expect(string(buf)).To(ContainSubstring("my-admin@example.com"))
			Expect(string(buf)).To(ContainSubstring("RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301,NE]"))

			procs, err := procmgr.ReadProcs(procsPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(procs.Processes).To(Equal(map[string]procmgr.Proc{
				"httpd": procmgr.Proc{
					Command: "httpd",
					Args:    []string{"-f", filepath.Join(factory.Build.Application.Root, "httpd.conf"), "-k", "start", "-DFOREGROUND"},
				},
			}))
		})
	})
}
