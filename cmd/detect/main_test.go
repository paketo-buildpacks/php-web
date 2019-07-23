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

package main

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/php-cnb/php"
	"github.com/cloudfoundry/php-web-cnb/phpweb"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is a PHP web app", func() {
		it("defaults `php.webdir` to `htdocs`", func() {
			Expect(pickWebDir(phpweb.BuildpackYAML{})).To(Equal("htdocs"))
		})

		it("loads `php.webdir` from `buildpack.yml`", func() {
			buildpackYAML := phpweb.BuildpackYAML{
				Config: phpweb.Config{
					WebDirectory: "public",
				},
			}

			Expect(pickWebDir(buildpackYAML)).To(Equal("public"))
		})

		it("finds a web app under `<webdir>/*.php`", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			found, err := searchForWebApp(factory.Detect.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("doesn't find a web app under `<webdir>/*.php`", func() {
			found, err := searchForWebApp(factory.Detect.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeFalse())
		})

		it("sets the proper buildplan items", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			factory.AddBuildPlan(php.Dependency, buildplan.Dependency{})
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
						"build":  true,
					},
					Version: fakeVersion,
				},
				"php-web": buildplan.Dependency{},
				"httpd": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
					},
				},
			}))
		})

		it("passes through Metadata.build", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			factory.AddBuildPlan(php.Dependency, buildplan.Dependency{
				Metadata: buildplan.Metadata{
					"build": true,
				},
			})
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
						"build":  true,
					},
					Version: fakeVersion,
				},
				"php-web": buildplan.Dependency{},
				"httpd": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
					},
				},
			}))
		})

		it("defaults php.webserver to apache webserver", func() {
			Expect(pickWebServer(phpweb.BuildpackYAML{})).To(Equal(httpd.Dependency))
		})

		it("will read php.webserver and select nginx", func() {
			Expect(pickWebServer(phpweb.BuildpackYAML{Config: phpweb.Config{WebServer: "nginx"}})).
				To(Equal("nginx"))
		})

		it("adjusts the buildplan webServer if non-default put in BuildpackYAML", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			factory.AddBuildPlan(php.Dependency, buildplan.Dependency{})
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), `{"php": {"webserver": "nginx"}}`)

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
						"build":  true,
					},
					Version: fakeVersion,
				},
				"php-web": buildplan.Dependency{},
				"nginx": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
					},
				},
			}))
		})

		it("fails if php-binary is not in the buildplan", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")

			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
		})
	})

	when("there is a PHP script", func() {
		it("finds a script in the root", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "main.php"), "")

			found, err := searchForScript(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("finds a script in a nested directory", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "test", "cli", "subdir", "my_cli.php"), "")

			found, err := searchForScript(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("doesn't find any script", func() {
			found, err := searchForScript(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeFalse())
		})

		it("sets the proper buildplan items", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "app.php"), "")
			factory.AddBuildPlan(php.Dependency, buildplan.Dependency{})
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
						"build":  true,
					},
					Version: fakeVersion,
				},
				"php-script": buildplan.Dependency{},
			}))
		})

		it("fails when there's no PHP files", func() {
			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
			Expect(factory.Output).To(BeNil())
		})
	})

	when("there is neither", func() {
		it("should fail", func() {
			factory.AddBuildPlan(php.Dependency, buildplan.Dependency{})
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}

			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
		})
	})
}
