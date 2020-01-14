/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package phpweb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/php-web-cnb/config"

	"github.com/cloudfoundry/php-web-cnb/procmgr"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitContributor(t *testing.T) {
	spec.Run(t, "Contributor", testContributor, spec.Report(report.Terminal{}))
}

func testContributor(t *testing.T, when spec.G, it spec.S) {
	var f *test.BuildFactory
	var CreateTestContributor func(bpYAML config.BuildpackYAML) Contributor

	CreateTestContributor = func(bpYAML config.BuildpackYAML) Contributor {
		bytes, err := yaml.Marshal(bpYAML)
		Expect(err).To(Not(HaveOccurred()))

		err = helper.WriteFile(filepath.Join(f.Build.Application.Root, "buildpack.yml"), 0644, string(bytes))
		Expect(err).To(Not(HaveOccurred()))

		c, _, err := NewContributor(f.Build)
		Expect(err).To(Not(HaveOccurred()))

		return c
	}

	it.Before(func() {
		RegisterTestingT(t)
		f = test.NewBuildFactory(t)

		f.AddPlan(buildpackplan.Plan{Name: Dependency})

		Expect(helper.WriteFile(filepath.Join(f.Build.Buildpack.Root, "bin", "procmgr"), os.ModePerm, "")).To(Succeed())
	})

	when("creating a new contributor", func() {
		it("generates random Metadata to prevent php-web layer from being cached", func() {
			c := CreateTestContributor(config.BuildpackYAML{})

			Expect(c.metadata.Name).To(Equal("PHP Web"))
			Expect(len(c.metadata.Hash)).To(Equal(64))
		})
	})

	when("starting a web app", func() {
		it.Before(func() {
			buildDir := filepath.Join(f.Build.Application.Root, "htdocs", "index.php")
			helper.WriteFile(buildDir, 0644, "junk")
		})

		it("starts a web app with HTTPD", func() {
			c := CreateTestContributor(config.BuildpackYAML{
				Config: config.Config{
					WebServer: config.ApacheHttpd,
				},
			})
			Expect(c.Contribute()).To(Succeed())

			phpLayer := f.Build.Layers.Layer(Dependency)
			procFile := filepath.Join(phpLayer.Root, "procs.yml")

			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
				Processes: []layers.Process{
					{Type: "web", Command: fmt.Sprintf("procmgr %s", procFile), Direct: false},
				},
			}))

			Expect(procFile).To(BeARegularFile())
			procs, err := procmgr.ReadProcs(procFile)
			Expect(err).ToNot(HaveOccurred())

			phpFpmProc := procmgr.Proc{
				Command: "php-fpm",
				Args:    []string{"-p", phpLayer.Root, "-y", filepath.Join(phpLayer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(phpLayer.Root, "etc")},
			}

			httpdProc := procmgr.Proc{
				Command: "httpd",
				Args:    []string{"-f", filepath.Join(f.Build.Application.Root, "httpd.conf"), "-k", "start", "-DFOREGROUND"},
			}

			Expect(procs.Processes).To(ContainElement(phpFpmProc))
			Expect(procs.Processes).To(ContainElement(httpdProc))
		})

		it("starts a web app and defaults to Apache Web Server", func() {
			test.WriteFile(t, filepath.Join(f.Build.Application.Root, "htdocs", "index.php"), "")

			c := CreateTestContributor(config.BuildpackYAML{
				Config: config.Config{
					WebServer: config.ApacheHttpd,
				},
			})
			Expect(c.Contribute()).To(Succeed())

			phpLayer := f.Build.Layers.Layer(Dependency)
			procFile := filepath.Join(phpLayer.Root, "procs.yml")

			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
				Processes: []layers.Process{
					{Type: "web", Command: fmt.Sprintf("procmgr %s", procFile), Direct: false},
				},
			}))

			Expect(procFile).To(BeARegularFile())

			file, err := os.Open(procFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			buf, err := ioutil.ReadAll(file)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(buf)).To(ContainSubstring("command: php-fpm"))
			Expect(string(buf)).To(ContainSubstring("-p"))
			Expect(string(buf)).To(ContainSubstring("layers/php-web"))
			Expect(string(buf)).To(ContainSubstring("-y"))
			Expect(string(buf)).To(ContainSubstring("layers/php-web/etc/php-fpm.conf"))
			Expect(string(buf)).To(ContainSubstring("-c"))
			Expect(string(buf)).To(ContainSubstring("layers/php-web/etc"))

			Expect(string(buf)).To(ContainSubstring("command: httpd"))
			Expect(string(buf)).To(ContainSubstring("-f"))
			Expect(string(buf)).To(ContainSubstring("application"))
			Expect(string(buf)).To(ContainSubstring("-k"))
			Expect(string(buf)).To(ContainSubstring("start"))
			Expect(string(buf)).To(ContainSubstring("-DFOREGROUND"))
		})

		it("starts a web app with NGINX", func() {
			c := CreateTestContributor(config.BuildpackYAML{
				Config: config.Config{
					WebServer: config.Nginx,
				},
			})

			Expect(c.Contribute()).To(Succeed())

			phpLayer := f.Build.Layers.Layer(Dependency)
			procFile := filepath.Join(phpLayer.Root, "procs.yml")

			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
				Processes: []layers.Process{
					{Type: "web", Command: fmt.Sprintf("procmgr %s", procFile), Direct: false},
				},
			}))

			Expect(procFile).To(BeARegularFile())
			procs, err := procmgr.ReadProcs(procFile)
			Expect(err).ToNot(HaveOccurred())

			phpFpmProc := procmgr.Proc{
				Command: "php-fpm",
				Args:    []string{"-p", phpLayer.Root, "-y", filepath.Join(phpLayer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(phpLayer.Root, "etc")},
			}

			nginxProc := procmgr.Proc{
				Command: "nginx",
				Args:    []string{"-p", f.Build.Application.Root, "-c", filepath.Join(f.Build.Application.Root, "nginx.conf")},
			}

			Expect(procs.Processes).To(ContainElement(phpFpmProc))
			Expect(procs.Processes).To(ContainElement(nginxProc))
		})

		when("the requested web server is not supported", func() {
			it("does not provide a start command", func() {
				c := CreateTestContributor(config.BuildpackYAML{
					Config: config.Config{
						WebServer: "notsupportedserver",
					},
				})
				Expect(c.Contribute()).To(Succeed())

				phpLayer := f.Build.Layers.Layer(Dependency)
				procFile := filepath.Join(phpLayer.Root, "procs.yml")
				Expect(procFile).ToNot(BeAnExistingFile())
			})
		})
	})

	when("contributing to build", func() {
		when("it's a web app", func() {
			it.Before(func() {
				buildDir := filepath.Join(f.Build.Application.Root, "htdocs", "index.php")
				helper.WriteFile(buildDir, 0644, "junk")
			})

			it("contributes a php.ini file & configures PHP to look at it for a web app", func() {
				c := CreateTestContributor(config.BuildpackYAML{
					Config: config.Config{
						WebServer: config.PhpWebServer,
					},
				})

				layer := f.Build.Layers.Layer(Dependency)
				Expect(c.Contribute()).To(Succeed())
				Expect(filepath.Join(layer.Root, "etc", "php.ini")).To(BeARegularFile())
				Expect(layer).To(test.HaveOverrideSharedEnvironment("PHPRC", filepath.Join(layer.Root, "etc")))
				Expect(layer).To(test.HaveOverrideSharedEnvironment("PHP_INI_SCAN_DIR", filepath.Join(f.Build.Application.Root, ".php.ini.d")))
			})

			it("contributes a httpd.conf & php-fpm.conf file when using Apache Web Server", func() {
				c := CreateTestContributor(config.BuildpackYAML{
					Config: config.Config{
						WebServer: config.ApacheHttpd,
					},
				})

				layer := f.Build.Layers.Layer(Dependency)
				Expect(c.Contribute()).To(Succeed())
				Expect(filepath.Join(f.Build.Application.Root, "httpd.conf")).To(BeARegularFile())
				Expect(filepath.Join(layer.Root, "etc", "php-fpm.conf")).To(BeARegularFile())
			})

			it("contributes a nginx.conf & php-fpm.conf file when using Nginx", func() {
				c := CreateTestContributor(config.BuildpackYAML{
					Config: config.Config{
						WebServer: config.Nginx,
					},
				})

				layer := f.Build.Layers.Layer(Dependency)
				Expect(c.Contribute()).To(Succeed())
				Expect(filepath.Join(f.Build.Application.Root, "nginx.conf")).To(BeARegularFile())
				Expect(filepath.Join(layer.Root, "etc", "php-fpm.conf")).To(BeARegularFile())
			})
		})

		when("it's not a web app", func() {
			it("contributes a php.ini file & configures PHP to look at it for a script", func() {
				c := CreateTestContributor(config.BuildpackYAML{})
				layer := f.Build.Layers.Layer(Dependency)
				Expect(c.Contribute()).To(Succeed())
				Expect(filepath.Join(layer.Root, "etc", "php.ini")).To(BeARegularFile())
				Expect(layer).To(test.HaveOverrideSharedEnvironment("PHPRC", filepath.Join(layer.Root, "etc")))
				Expect(layer).To(test.HaveOverrideSharedEnvironment("PHP_INI_SCAN_DIR", filepath.Join(f.Build.Application.Root, ".php.ini.d")))
			})
		})
	})
}
