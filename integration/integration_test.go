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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)

	err := PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())

	spec.Run(t, "Online", testIntegration, spec.Report(report.Terminal{}))
	spec.Run(t, "Offline", testOffline, spec.Report(report.Terminal{}), spec.Parallel())
	CleanUpBps()
}

func HTTPGetLikeProxy(app *dagger.App, path string) (string, map[string][]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", app.GetBaseURL(), path), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Add("X-Forwarded-For", "10.10.10.10,50.50.50.50")
	req.Header.Add("X-Forwarded-Proto", "http")
	req.Header.Add("X-Forwarded-Port", "80")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("received bad response from application")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	return string(body), resp.Header, nil
}

func AssertRedirectToHTTPS(app *dagger.App, headers map[string][]string) {
	for key, header := range headers {
		for _, value := range header {
			if key == "Location" {
				Expect(value).To(ContainSubstring(strings.Replace(app.GetBaseURL(), "http://", "https://", 1)))
			}
		}
	}
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app    *dagger.App
		err    error
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	when("deploying the simple_app fixture", func() {
		it("serves a simple php page hosted with built-in PHP server as the default", func() {
			app, err = PushSimpleApp("simple_app_php_only", []string{phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- PHP Web Server"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))
		})

		it("serves a simple php page with httpd", func() {
			app, err = PushSimpleApp("simple_app_httpd", []string{httpdURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Apache Web Server"))
			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("web: procmgr /layers/%s/php-web/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))))
			Expect(app.BuildLogs()).To(MatchRegexp(`Apache HTTP Server Buildpack (v?)\d+.\d+.\d+`))
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))

			_, headers, err := HTTPGetLikeProxy(app, "/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			AssertRedirectToHTTPS(app, headers)
		})

		it("serves a simple php page with httpd and custom httpd config", func() {
			app, err = PushSimpleApp("simple_app_custom_httpd_cfg", []string{httpdURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Apache Web Server"))
			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("web: procmgr /layers/%s/php-web/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))))
			Expect(app.BuildLogs()).To(MatchRegexp(`Apache HTTP Server Buildpack (v?)\d+.\d+.\d+`))
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

			resp, _, err := app.HTTPGet("/status?auto")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("ServerMPM: event"))
		})

		it("serves a simple php page with nginx", func() {
			app, err = PushSimpleApp("simple_app_nginx", []string{nginxURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Nginx"))
			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("web: procmgr /layers/%s/php-web/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))))
			Expect(app.BuildLogs()).To(MatchRegexp(`Installing Nginx Server \d+.\d+.\d+`))
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))

			_, headers, err := HTTPGetLikeProxy(app, "/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			AssertRedirectToHTTPS(app, headers)
		})

		it("serves a simple php page with nginx and custom config", func() {
			app, err = PushSimpleApp("simple_app_nginx_custom_cfg", []string{nginxURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Nginx"))
			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("web: procmgr /layers/%s/php-web/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))))
			Expect(app.BuildLogs()).To(MatchRegexp(`Installing Nginx Server \d+.\d+.\d+`))
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

			// changed in custom-http.conf
			resp, headers, err := app.HTTPGet("/test.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))

			// changed in custom-server.conf
			serverHeader, found := headers["Server"]
			Expect(found).To(BeTrue())
			Expect(len(serverHeader)).To(Equal(1))
			Expect(serverHeader[0]).To(MatchRegexp(`^nginx/\d+\.\d+\.\d+`))
		})

		it("runs a cli app", func() {
			app, err = PushSimpleApp("simple_cli_app", []string{phpDistURI, phpWebURI}, true)
			Expect(err).NotTo(HaveOccurred())

			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("SUCCESS"))
		})

		it("runs a cli app with arguments", func() {
			app, err := PushSimpleApp("simple_cli_app_with_args", []string{phpDistURI, phpWebURI}, true)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.script -> use a Procfile"))
			logs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).To(ContainSubstring("ALTERNATE"))
		})
	})

	when("deploying a basic PHP app with extensions", func() {
		it("loads list of expected extensions", func() {
			app, err = PreparePhpApp("php_modules", []string{phpDistURI, phpWebURI}, nil)
			Expect(err).ToNot(HaveOccurred())
			app.SetHealthCheck("true", "3s", "1s")
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))
			err := app.Start()

			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
				Expect(err).NotTo(HaveOccurred())

				containerID, imageName, volumeIDs, err := app.Info()
				Expect(err).NotTo(HaveOccurred())

				fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

				containerLogs, err := app.Logs()
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("Container Logs:\n %s\n", containerLogs)
				t.FailNow()
			}

			output, err := app.Logs()
			Expect(err).NotTo(HaveOccurred())
			Expect(output).ToNot(ContainSubstring("Unable to load dynamic library"), app.BuildLogs)

			for _, extension := range ExpectedExtensions {
				Expect(output).To(ContainSubstring(extension))
			}
		})
	})

	when("deploying the php_app fixture", func() {
		it("does not return the version of PHP in the response headers", func() {
			app, err = PreparePhpApp("php_app", []string{phpDistURI, phpWebURI}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

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

			// ensure X-Powered-By header is removed so as not to leak information
			body, headers, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("PHP Version"))
			Expect(headers).ToNot(HaveKey("X-Powered-By"))
		})

		it("installs our hard-coded default version of PHP", func() {
			app, err = PreparePhpApp("php_app", []string{phpDistURI, phpWebURI}, nil)
			Expect(err).ToNot(HaveOccurred())

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
			Expect(app.BuildLogs()).To(MatchRegexp(`Installing PHP 7\.4\.\d+`))
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))
		})

		when("the app is pushed twice", func() {
			it("does generate php config twice", func() {
				appName := "php_app"
				env := make(map[string]string)
				env["BP_DEBUG"] = "true"

				app, err = PreparePhpApp(appName, []string{phpDistURI, phpWebURI}, env)
				Expect(err).ToNot(HaveOccurred())

				Expect(app.BuildLogs()).To(MatchRegexp("PHP Web .*: Contributing to layer"))

				app, err = dagger.PackBuildNamedImageWithEnv(app.ImageName, filepath.Join("testdata", appName), env, []string{phpDistURI, phpWebURI}...)

				Expect(app.BuildLogs()).To(MatchRegexp("PHP Web .*: Contributing to layer"))
				Expect(app.BuildLogs()).NotTo(MatchRegexp("PHP Web .*: Reusing cached layer"))

				Expect(app.Start()).To(Succeed())
			})
		})
	})

	when("deploying an app with sessions", func() {
		it("redis session support is enabled and data is stored in redis", func() {
			env := make(map[string]string)
			env["CNB_SERVICES"] = `{
				"Services": [
					{
						"binding_name": "redis-sessions",
						"credentials": {
							"host": "host.docker.internal",
							"port": 63009
						},
						"instance_name": "",
						"label": "",
						"plan": "",
						"tags": null
					}
				]
			}`

			app, err = PreparePhpApp("session_test", []string{phpDistURI, phpWebURI}, env)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.BuildLogs()).To(MatchRegexp(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			Expect(app.BuildLogs()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(app.BuildLogs()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())

			Expect(body).To(ContainSubstring("Redis Loaded: 1"))
			Expect(body).To(ContainSubstring("Session Handler: redis"))
			Expect(body).To(ContainSubstring("Session Name: PHPSESSIONID"))
			Expect(body).To(ContainSubstring("Session Save Path: tcp://host.docker.internal:63009"))

			_, _, err = app.HTTPGet("/session.php")
			Expect(err).ToNot(HaveOccurred())
			appLogs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(appLogs).To(ContainSubstring("PHP Notice:  session_start(): Redis not available while creating session_id"))
			Expect(appLogs).To(ContainSubstring("PHP Warning:  session_start(): Failed to read session data"))
		})

		it("memcached session support is enabled and data is stored in memcached", func() {
			env := make(map[string]string)
			env["CNB_SERVICES"] = `{
				"Services": [
					{
						"binding_name": "memcached-sessions",
						"credentials": {
							"servers": "host.docker.internal:60039",
							"username": "user-1",
							"password": "passwoRd"
						},
						"instance_name": "",
						"label": "",
						"plan": "",
						"tags": null
					}
				]
			}`

			app, err = PreparePhpApp("session_test", []string{phpDistURI, phpWebURI}, env)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())

			Expect(body).To(ContainSubstring("Memcached Loaded: 1"))
			Expect(body).To(ContainSubstring("Session Handler: memcached"))
			Expect(body).To(ContainSubstring("Session Name: PHPSESSIONID"))
			Expect(body).To(ContainSubstring("Session Save Path: host.docker.internal:60039"))
			Expect(body).To(ContainSubstring("Memcached Session Binary: 1"))
			Expect(body).To(ContainSubstring("Memcached SASL User: user-1"))
			Expect(body).To(ContainSubstring("Memcached SASL Pass: passwoRd"))

			_, _, err = app.HTTPGet("/session.php")
			Expect(err).To(HaveOccurred())
			appLogs, err := app.Logs()
			Expect(err).ToNot(HaveOccurred())
			Expect(appLogs).To(ContainSubstring("PHP Fatal error:  Uncaught Error: Failed to create session ID: memcached"))
			Expect(appLogs).To(ContainSubstring("/session.php - Uncaught Error: Failed to create session ID: memcached"))
		})
	})
}

// NOTE: as extensions are added to the php-dist-cnb binaries, we need to update this list
//   and also integration/testdata/php_modules/.php.ini.d/snippet.ini
var ExpectedExtensions = [...]string{
	"amqp",
	"apcu",
	"bz2",
	"curl",
	"dba",
	"enchant",
	"exif",
	"fileinfo",
	"ftp",
	"gd",
	"gettext",
	"gmp",
	"igbinary",
	"imagick",
	"imap",
	"ionCube",
	"ldap",
	"lua",
	"lzf",
	"mailparse",
	"maxminddb",
	"mbstring",
	"memcached",
	"mongodb",
	"msgpack",
	"mysqli",
	"OAuth",
	"OPcache",
	"openssl",
	"pcntl",
	"PDO",
	"PDO_Firebird",
	"pdo_mysql",
	"PDO_ODBC",
	"pdo_pgsql",
	"pdo_sqlite",
	"pdo_sqlsrv",
	"pgsql",
	"phpiredis",
	"protobuf",
	"pspell",
	"psr",
	"rdkafka",
	"readline",
	"redis",
	"shmop",
	"snmp",
	"soap",
	"sockets",
	"sodium",
	"solr",
	"sqlsrv",
	"ssh2",
	"Stomp",
	"sysvmsg",
	"sysvsem",
	"sysvshm",
	"tideways_xhprof",
	"tidy",
	"xdebug",
	"xmlrpc",
	"xsl",
	"yaml",
	"zip",
	"zlib",
}
