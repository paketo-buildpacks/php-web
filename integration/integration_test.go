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

var (
	app *dagger.App
	err error
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)

	var err error
	err = PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}), spec.Parallel())
	CleanUpBps()
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
		app.Destroy()
	})

	when("deploying the simple_app fixture", func() {
		it("serves a simple php page with httpd", func() {
			app, err = PushSimpleApp("simple_app", []string{httpdURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Apache Web Server"))
			Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))
			Expect(app.BuildLogs()).To(MatchRegexp("Apache HTTP Server .*: Contributing to layer"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))
		})

		it("serves a simple php page with httpd and custom httpd config", func() {
			app, err = PushSimpleApp("simple_app_custom_httpd_cfg", []string{httpdURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Apache Web Server"))
			Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))
			Expect(app.BuildLogs()).To(MatchRegexp("Apache HTTP Server .*: Contributing to layer"))

			resp, _, err := app.HTTPGet("/status?auto")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("ServerMPM: event"))
		})

		it("serves a simple php page hosted with built-in PHP server", func() {
			app, err = PushSimpleApp("simple_app_php_only", []string{phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- PHP Web Server"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))
		})

		it("serves a simple php page with nginx", func() {
			app, err = PushSimpleApp("simple_app_nginx", []string{nginxURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Nginx"))
			Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))
			Expect(app.BuildLogs()).To(MatchRegexp("Nginx Server .*: Contributing to layer"))

			resp, _, err := app.HTTPGet("/index.php?date")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring("SUCCESS"))
		})

		it("serves a simple php page with nginx and custom config", func() {
			app, err = PushSimpleApp("simple_app_nginx_custom_cfg", []string{nginxURI, phpDistURI, phpWebURI}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(app.BuildLogs()).To(ContainSubstring("Using feature -- Nginx"))
			Expect(app.BuildLogs()).To(ContainSubstring("web: procmgr /layers/org.cloudfoundry.php-web/php-web/procs.yml"))
			Expect(app.BuildLogs()).To(MatchRegexp("Nginx Server .*: Contributing to layer"))

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
			defer app.Destroy()

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
			err := app.Start()

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

			output, err := app.Logs()

			Expect(output).ToNot(ContainSubstring("Unable to load dynamic library"))

			for _, extension := range ExpectedExtensions {
				Expect(output).To(ContainSubstring(extension))
			}
		})
	})

	when("deploying the php_app fixture", func() {
		it("does not return the version of PHP in the response headers", func() {
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
			Expect(app.BuildLogs()).To(MatchRegexp(`PHP.*7\.2\.\d+.*Contributing.* to layer`))
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
	})
}

// NOTE: as extensions are added to the php-dist-cnb binaries, we need to update this list
//   and also integration/testdata/php_modules/.php.ini.d/snippet.ini
var ExpectedExtensions = [...]string{
	"bz2",
	"curl",
	"dba",
	"exif",
	"fileinfo",
	"ftp",
	"gd",
	"gettext",
	"gmp",
	"imap",
	"ldap",
	"mbstring",
	"mysqli",
	"Zend OPcache",
	"openssl",
	"pcntl",
	"pdo",
	"pdo_mysql",
	"pdo_pgsql",
	"pdo_sqlite",
	"pgsql",
	"pspell",
	"shmop",
	"snmp",
	"soap",
	"sockets",
	"sysvmsg",
	"sysvsem",
	"sysvshm",
	"xsl",
	"zip",
	"zlib",
	"apcu",
	"cassandra",
	"geoip",
	"igbinary",
	"gnupg",
	"imagick",
	"lzf",
	"mailparse",
	"mongodb",
	"msgpack",
	"OAuth",
	"odbc",
	"PDO_ODBC",
	"pdo_sqlsrv",
	"rdkafka",
	"redis",
	"sqlsrv",
	"Stomp",
	"xdebug",
	"yaf",
	"yaml",
	"memcached",
	"sodium",
	"tidy",
	"enchant",
	"interbase",
	"PDO_Firebird",
	"readline",
	"wddx",
	"xmlrpc",
	"recode",
	"amqp",
	"lua",
	"maxminddb",
	"phalcon",
	"phpiredis",
	"protobuf",
	"tideways",
	"tideways_xhprof",
	"ionCube Loader",
}
