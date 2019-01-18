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
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/php-cnb/php"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		factory = test.NewDetectFactory(t)
	})

	when("there is a PHP web app", func() {
		it("defaults `php.webdir` to `htdocs`", func() {
			Expect(pickWebDir(php.BuildpackYAML{})).To(Equal("htdocs"))
		})

		it("loads `php.webdir` from `buildpack.yml`", func() {
			buildpackYAML := php.BuildpackYAML{
				Config: php.Config{
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

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
					},
				},
				"php-web": buildplan.Dependency{},
			}))
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

			Expect(runDetect(factory.Detect)).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				"php-binary": buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"launch": true,
					},
				},
				"php-script": buildplan.Dependency{},
			}))
		})

		it("fails when there's no PHP files", func() {
			Expect(runDetect(factory.Detect)).To(Equal(detect.FailStatusCode))
			Expect(factory.Output).To(BeNil())
		})
	})
}
