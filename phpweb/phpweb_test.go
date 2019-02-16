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
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/php-cnb/php"

	bp "github.com/buildpack/libbuildpack/buildpack"
	"github.com/buildpack/libbuildpack/buildplan"
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

	when("a version is set", func() {
		it("uses php.version from buildpack.yml, if set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{}, logger.Logger{})
			dependency := buildplan.Dependency{}
			buildpackYAML := BuildpackYAML{
				Config: Config{
					Version: "test-version",
				},
			}

			Expect(Version(buildpackYAML, buildpack, dependency)).To(Equal("test-version"))
		})

		it("uses build plan version, if set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{}, logger.Logger{})
			dependency := buildplan.Dependency{Version: "test-version"}

			Expect(Version(BuildpackYAML{}, buildpack, dependency)).To(Equal("test-version"))
		})

		it("uses buildpack default version if set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{Metadata: buildpack.Metadata{"default_version": "test-version"}}, logger.Logger{})
			dependency := buildplan.Dependency{}

			Expect(Version(BuildpackYAML{}, buildpack, dependency)).To(Equal("test-version"))
		})

		it("return `*` if none set", func() {
			buildpack := buildpack.NewBuildpack(bp.Buildpack{}, logger.Logger{})
			dependency := buildplan.Dependency{}

			Expect(Version(BuildpackYAML{}, buildpack, dependency)).To(Equal("*"))
		})

	})

	when("buildpack.yml", func() {
		var f *test.DetectFactory

		it.Before(func() {
			f = test.NewDetectFactory(t)
		})

		it("can load an empty buildpack.yaml", func() {
			test.WriteFile(t, filepath.Join(f.Detect.Application.Root, "buildpack.yml"), "")

			loaded, err := LoadBuildpackYAML(f.Detect.Application.Root)

			Expect(err).To(Succeed())
			Expect(loaded).To(Equal(BuildpackYAML{}))
		})

		it("can load a version & web server", func() {
			yaml := "{'php': {'version': 1.0.0, 'webserver': 'httpd'}}"
			test.WriteFile(t, filepath.Join(f.Detect.Application.Root, "buildpack.yml"), yaml)

			loaded, err := LoadBuildpackYAML(f.Detect.Application.Root)
			actual := BuildpackYAML{
				Config: Config{
					Version:   "1.0.0",
					WebServer: "httpd",
				},
			}

			Expect(err).To(Succeed())
			Expect(loaded).To(Equal(actual))
		})
	})

	when("we need the api string", func() {
		it("converts from version number", func() {
			Expect("20151012").To(Equal(API("7.0.1")))
			Expect("20151012").To(Equal(API("7.0")))
			Expect("20160303").To(Equal(API("7.1.25")))
			Expect("20160303").To(Equal(API("7.1")))
			Expect("20170718").To(Equal(API("7.2.15")))
			Expect("20170718").To(Equal(API("7.2")))
			Expect("20180731").To(Equal(API("7.3.1")))
			Expect("20180731").To(Equal(API("7.3")))
		})
	})

	when("we need a list of PHP extensions", func() {
		var f *test.BuildFactory

		it.Before(func() {
			f = test.NewBuildFactory(t)
		})

		it("loads the available extensions", func() {
			layer := f.Build.Layers.Layer(php.Dependency)
			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "dba.so"), "")
			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "curl.so"), "")
			test.WriteFile(t, filepath.Join(layer.Root, "lib", "php", "extensions", "no-debug-non-zts-20170718", "mysqli.so"), "")

			extensions, err := LoadAvailablePHPExtensions(layer.Root, "7.2")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(extensions)).To(Equal(3))
		})
	})

	when("we have metadata", func() {
		var f *test.BuildFactory

		it.Before(func() {
			f = test.NewBuildFactory(t)
		})

		it("has a name & version", func() {
			md := NewMetadata("1.0")

			Expect(md.Name).To(Equal("PHP Web"))
			Expect(md.BuildpackVersion).To(Equal("1.0"))
			Expect(md.PhpFpmUserConfig).To(BeFalse())
		})

		it("has a hash", func() {
			buildpackYAMLPath := filepath.Join(f.Build.Application.Root, "buildpack.yml")
			test.WriteFile(t, buildpackYAMLPath, "Hello World!")

			md := NewMetadata("1.0")
			md.UpdateHashFromFile(buildpackYAMLPath)

			Expect(md.BuildpackYAMLHash).To(Equal("7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"))
		})

		it("has a hash of the date when buildpack.yml doesn't exist", func() {
			md := NewMetadata("1.0")
			md.UpdateHashFromFile(filepath.Join(f.Build.Application.Root, "buildpack.yml"))

			firstHash := md.BuildpackYAMLHash

			time.Sleep(10 * time.Millisecond)

			md.UpdateHashFromFile(filepath.Join(f.Build.Application.Root, "buildpack.yml"))

			Expect(firstHash).ToNot(Equal(md.BuildpackYAMLHash))
		})

		it("generates an identity", func() {
			buildpackYAMLPath := filepath.Join(f.Build.Application.Root, "buildpack.yml")
			test.WriteFile(t, buildpackYAMLPath, "Hello World!")

			md := NewMetadata("1.0")
			md.UpdateHashFromFile(buildpackYAMLPath)
			md.PhpFpmUserConfig = true

			name, version := md.Identity()

			Expect(name).To(Equal("PHP Web"))
			Expect(version).To(Equal("1.0:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069:true"))
		})
	})

	when("user provides PHP-FPM config", func() {
		var f *test.BuildFactory

		it.Before(func() {
			f = test.NewBuildFactory(t)
		})

		it("detects the folder path", func() {
			test.WriteFile(t, filepath.Join(f.Build.Application.Root, ".php.fpm.d", "user.conf"), "")

			path, err := GetPhpFpmConfPath(f.Build.Application.Root)

			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(f.Build.Application.Root, ".php.fpm.d", "*.conf")))
		})

		it("returns nothing if there are no files", func() {
			path, err := GetPhpFpmConfPath(f.Build.Application.Root)

			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(BeEmpty())
		})
	})
}
