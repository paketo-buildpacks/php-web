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

	"github.com/buildpack/libbuildpack/buildpack"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var builderMetadata dagger.BuilderMetadata
	var builderMetadataWithHttpd dagger.BuilderMetadata

	it.Before(func() {
		RegisterTestingT(t)
		uri, err := dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())

		phpBpURI, err := dagger.GetRemoteBuildpack("https://github.com/cloudfoundry/php-cnb/releases/download/v0.0.1/php-cnb-0.0.1.tgz")
		Expect(err).ToNot(HaveOccurred())

		httpdBpURI, err := dagger.GetRemoteBuildpack("https://github.com/cloudfoundry/httpd-cnb/releases/download/v0.0.1/httpd-cnb-0.0.1.tgz")
		Expect(err).ToNot(HaveOccurred())

		builderMetadata = dagger.BuilderMetadata{
			Buildpacks: []dagger.Buildpack{
				{
					ID:  "org.cloudfoundry.buildpacks.php",
					URI: phpBpURI,
				},
				{
					ID:  "org.cloudfoundry.buildpacks.php-web",
					URI: uri,
				},
			},
			Groups: []dagger.Group{
				{
					[]buildpack.Info{
						{
							ID:      "org.cloudfoundry.buildpacks.php",
							Version: "0.0.1",
						},
						{
							ID:      "org.cloudfoundry.buildpacks.php-web",
							Version: "0.0.1",
						},
					},
				},
			},
		}

		builderMetadataWithHttpd = dagger.BuilderMetadata{
			Buildpacks: []dagger.Buildpack{
				{
					ID:  "org.cloudfoundry.buildpacks.php",
					URI: phpBpURI,
				},
				{
					ID:  "org.cloudfoundry.buildpacks.php-web",
					URI: uri,
				},
				{
					ID:  "org.cloudfoundry.buildpacks.httpd",
					URI: httpdBpURI,
				},
			},
			Groups: []dagger.Group{
				{
					[]buildpack.Info{
						{
							ID:      "org.cloudfoundry.buildpacks.php",
							Version: "0.0.1",
						},
						{
							ID:      "org.cloudfoundry.buildpacks.httpd",
							Version: "0.0.1",
						},
						{
							ID:      "org.cloudfoundry.buildpacks.php-web",
							Version: "0.0.1",
						},
					},
				},
			},
		}
	})

	when("push simple app", func() {
		it("servers simple php page", func() {
			app, err := dagger.Pack(filepath.Join("fixtures", "simple_app"), builderMetadataWithHttpd, dagger.CFLINUXFS3)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")
			app.Env["PORT"] = "8080"

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.ContainerInfo()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.ContainerLogs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			resp, err := app.HTTPGetSucceeds("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp)).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed()) //Only destroy app if the test passed to leave artifacts to debug
		})

		it("servers simple php page hosted with built-in PHP server", func() {
			app, err := dagger.Pack(filepath.Join("fixtures", "simple_app_php_only"), builderMetadata, dagger.CFLINUXFS3)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")
			app.Env["PORT"] = "8080"

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.ContainerInfo()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.ContainerLogs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			resp, err := app.HTTPGetSucceeds("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp)).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed()) //Only destroy app if the test passed to leave artifacts to debug
		})

		it("runs a cli app", func() {
			app, err := dagger.Pack(filepath.Join("fixtures", "simple_cli_app"), builderMetadata, dagger.CFLINUXFS3)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("true", "3s", "1s") // disables health check
			app.Env["PORT"] = "8080"
			// TODO add DisableHealthCheck to dagger

			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				containerID, imageName, volumeIDs, err := app.ContainerInfo()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.ContainerLogs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			logs, err := app.ContainerLogs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed()) //Only destroy app if the test passed to leave artifacts to debug
		})
	})
}
