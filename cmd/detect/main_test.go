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

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/phpweb"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
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
		it("sets the proper buildplan items", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Requires: []buildplan.Required{
					{
						Name:    "php",
						Version: fakeVersion,
						Metadata: buildplan.Metadata{"launch": true, "build": true,
							buildpackplan.VersionSource: "default-versions"},
					},
					{Name: phpweb.Dependency},
					{
						Name:     "php-server",
						Metadata: buildplan.Metadata{"launch": true},
					},
				},
				Provides: []buildplan.Provided{
					{Name: phpweb.Dependency}, {Name: config.PhpWebServer},
				},
			}))
		})

		it("passes through Metadata.build", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Requires: []buildplan.Required{
					{
						Name:    "php",
						Version: fakeVersion,
						Metadata: buildplan.Metadata{"launch": true, "build": true,
							buildpackplan.VersionSource: "default-versions"},
					},
					{Name: phpweb.Dependency},
					{
						Name:     "php-server",
						Metadata: buildplan.Metadata{"launch": true},
					},
				},
				Provides: []buildplan.Provided{
					{Name: phpweb.Dependency}, {Name: config.PhpWebServer},
				},
			}))
		})

		it("defaults php.webserver to apache webserver", func() {
			Expect(pickWebServer(config.BuildpackYAML{})).To(Equal("php-server"))
		})

		it("will read php.webserver and select nginx", func() {
			Expect(pickWebServer(config.BuildpackYAML{Config: config.Config{WebServer: "nginx"}})).
				To(Equal("nginx"))
		})

		it("adjusts the buildplan webServer if non-default put in BuildpackYAML", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), `{"php": {"webserver": "nginx"}}`)

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Requires: []buildplan.Required{
					{
						Name:    "php",
						Version: fakeVersion,
						Metadata: buildplan.Metadata{"launch": true, "build": true,
							buildpackplan.VersionSource: "default-versions"},
					},
					{Name: phpweb.Dependency},
					{
						Name:     "nginx",
						Metadata: buildplan.Metadata{"launch": true},
					},
				},
				Provides: []buildplan.Provided{
					{Name: phpweb.Dependency},
				},
			}))
		})
	})

	when("there is a PHP script", func() {
		it("finds a script in the root", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "main.php"), "")

			found, err := searchForAnyPHPFiles(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("finds a script in a nested directory", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "test", "cli", "subdir", "my_cli.php"), "")

			found, err := searchForAnyPHPFiles(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("doesn't find any script", func() {
			found, err := searchForAnyPHPFiles(factory.Detect.Application.Root, factory.Detect.Logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeFalse())
		})

		it("sets the proper buildplan items", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "app.php"), "")
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Requires: []buildplan.Required{
					{
						Name:    "php",
						Version: fakeVersion,
						Metadata: buildplan.Metadata{"launch": true, "build": true,
							buildpackplan.VersionSource: "default-versions"},
					},
					{Name: phpweb.Dependency},
				},
				Provides: []buildplan.Provided{
					{Name: phpweb.Dependency},
				},
			}))
		})

		it("fails when there's no PHP files", func() {
			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
			Expect(factory.Plans.Plan.Requires).To(BeNil())
			Expect(factory.Plans.Plan.Provides).To(BeNil())
		})
	})

	when("there is neither", func() {
		it("should fail", func() {
			fakeVersion := "php.default.version"
			factory.Detect.Buildpack.Metadata = map[string]interface{}{"default_version": fakeVersion}

			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
		})
	})
}
