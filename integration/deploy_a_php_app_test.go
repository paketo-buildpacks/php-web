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

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestDeployAPHPAppIntegration(t *testing.T) {
	RegisterTestingT(t)

	var err error
	buildpacks, err = PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		for _, buildpack := range buildpacks {
			dagger.DeleteBuildpack(buildpack)
		}
	}()

	spec.Run(t, "Deploy A PHP App", testDeployAPHPAppIntegration, spec.Report(report.Terminal{}))
}

func testDeployAPHPAppIntegration(t *testing.T, when spec.G, it spec.S) {
	var app *dagger.App
	var err error

	it.Before(func() {
		RegisterTestingT(t)
	})

	when("deploying a basic PHP app", func() {
		it("installs our hard-coded default version of PHP and does not return the version of PHP in the response headers", func() {
			app, err = PreparePhpApp("php_app", buildpacks, false)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			// ensure correct version of PHP is installed
			Expect(app.BuildLogs()).To(MatchRegexp(`PHP.*7\.2\.\d+.*Contributing.* to layer`))

			// ensure X-Powered-By header is removed so as not to leak information
			body, headers, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("PHP Version"))
			Expect(headers).ToNot(HaveKey("X-Powered-By"))
		})

		when("the app is pushed twice", func() {
			it("does not generate php config twice", func() {
				appName := "php_app"
				debug := false
				app, err := PreparePhpApp(appName, buildpacks, false)
				Expect(err).ToNot(HaveOccurred())
				defer app.Destroy()

				Expect(app.BuildLogs()).To(MatchRegexp("PHP Web .*: Contributing to layer"))
				Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))

				app, err = dagger.PackBuildNamedImageWithEnv(app.ImageName, filepath.Join("testdata", appName), MakeBuildEnv(debug), buildpacks...)

				Expect(app.BuildLogs()).To(MatchRegexp("PHP Web .*: Contributing to layer"))
				Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))
				Expect(app.BuildLogs()).NotTo(MatchRegexp("PHP Web .*: Reusing cached layer"))

				Expect(app.Start()).To(Succeed())

				// ensure X-Powered-By header is removed so as not to leak information
				body, headers, err := app.HTTPGet("/")
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(ContainSubstring("PHP Version"))
				Expect(headers).ToNot(HaveKey("X-Powered-By"))
			})
		})
	})
}
