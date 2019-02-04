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
	"github.com/cloudfoundry/dagger"
	"os"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		uri, phpBpURI, httpdBpURI string
	)

	it.Before(func() {
		var err error

		RegisterTestingT(t)
		uri, err = dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())

		phpBpURI, err = dagger.GetLatestBuildpack("php-cnb")
		Expect(err).ToNot(HaveOccurred())

		httpdBpURI, err = dagger.GetLatestBuildpack("httpd-cnb")
		Expect(err).ToNot(HaveOccurred())
	})

	when("push simple app", func() {
		it("servers simple php page", func() {
			app, err := dagger.PackBuild(filepath.Join("fixtures", "simple_app"), phpBpURI, httpdBpURI, uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")
			app.Env["PORT"] = "8080"

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

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed())
		})

		it("servers simple php page hosted with built-in PHP server", func() {
			app, err := dagger.PackBuild(filepath.Join("fixtures", "simple_app_php_only"), phpBpURI, uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")
			app.Env["PORT"] = "8080"

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

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed())
		})

		it("runs a cli app", func() {
			app, err := dagger.PackBuild(filepath.Join("fixtures", "simple_cli_app"), phpBpURI, uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("true", "3s", "1s") // disables health check
			app.Env["PORT"] = "8080"
			// TODO add DisableHealthCheck to dagger

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

			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))

			Expect(app.Destroy()).To(Succeed())
		})
	})
}
