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
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/php-dist-cnb/php"
	"github.com/cloudfoundry/php-web-cnb/config"

	bp "github.com/buildpack/libbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPHPWeb(t *testing.T) {
	spec.Run(t, "PHPWeb", testPHPWeb, spec.Report(report.Terminal{}))
}

func testPHPWeb(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("checking for a web app", func() {
		var factory *test.DetectFactory

		it.Before(func() {
			factory = test.NewDetectFactory(t)
		})

		it("defaults `php.webdir` to `htdocs`", func() {
			Expect(PickWebDir(config.BuildpackYAML{})).To(Equal("htdocs"))
		})

		it("loads `php.webdirectory` from `buildpack.yml`", func() {
			buildpackYAML := config.BuildpackYAML{
				Config: config.Config{
					WebDirectory: "public",
				},
			}

			Expect(PickWebDir(buildpackYAML)).To(Equal("public"))
		})

		it("finds a web app under `<webdir>/*.php`", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "htdocs", "index.php"), "")
			found, err := SearchForWebApp(factory.Detect.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("doesn't find a web app under `<webdir>/*.php`", func() {
			found, err := SearchForWebApp(factory.Detect.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeFalse())
		})

	})

	when("a version is set", func() {
		it("uses buildpack default version if set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{Metadata: buildpack.Metadata{"default_version": "test-version"}}, logger.Logger{})

			Expect(Version(buildpack)).To(Equal("test-version"))
		})

		it("return `*` if none set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{}, logger.Logger{})

			Expect(Version(buildpack)).To(Equal("*"))
		})

	})

	when("we need a list of PHP extensions", func() {
		var f *test.BuildFactory

		it.Before(func() {
			f = test.NewBuildFactory(t)
		})

		it("loads the available extensions", func() {
			layer := f.Build.Layers.Layer(php.Dependency)

			// WARN: this is setting a global env variable, which might cause issues if tests are run in parallel
			os.Setenv("PHP_EXTENSION_DIR", filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718"))

			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "dba.so"), "")
			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "curl.so"), "")
			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "mysqli.so"), "")

			extensions, err := LoadAvailablePHPExtensions()
			fmt.Println("extensions:", extensions)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(extensions)).To(Equal(3))
		})
	})
}
