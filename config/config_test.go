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

package config

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	bp "github.com/buildpack/libbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPhpAppConfig(t *testing.T) {
	spec.Run(t, "Httpd", testPhpAppConfig, spec.Report(report.Terminal{}))
}

func testPhpAppConfig(t *testing.T, when spec.G, it spec.S) {
	var f *test.BuildFactory

	it.Before(func() {
		RegisterTestingT(t)
		f = test.NewBuildFactory(t)
	})

	when("config generation", func() {
		it("generates an httpd.conf from the template", func() {
			cfg := HttpdConfig{
				AppRoot:      "/app",
				ServerAdmin:  "test@example.org",
				WebDirectory: "htdocs",
				FpmSocket:    "127.0.0.1:9000",
			}

			err := ProcessTemplateToFile(HttpdConfTemplate, filepath.Join(f.Home, "httpd.conf"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "httpd.conf"))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`ServerRoot "${SERVER_ROOT}"`))
			Expect(result).To(ContainSubstring(`ServerAdmin "test@example.org"`))
			Expect(result).To(ContainSubstring(`DocumentRoot "/app/htdocs"`))
			Expect(result).To(ContainSubstring(`<Directory "/app/htdocs">`))
			Expect(result).To(ContainSubstring(`<FilesMatch "^\.">`))
			Expect(result).To(ContainSubstring(`ErrorLog "/proc/self/fd/2"`))
			Expect(result).To(ContainSubstring(`CustomLog "/proc/self/fd/1" extended`))
			Expect(result).To(ContainSubstring(`RemoteIpHeader x-forwarded-for`))
			Expect(result).To(ContainSubstring(`RemoteIpInternalProxy 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16`))
			Expect(result).To(ContainSubstring(`SetEnvIf x-forwarded-proto https HTTPS=on`))
			Expect(result).To(ContainSubstring(`RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301,NE]`))
			Expect(result).To(ContainSubstring(`Define fcgi-listener fcgi://127.0.0.1:9000/app/htdocs`))
			Expect(result).To(ContainSubstring(`<Proxy "${fcgi-listener}">`))
			Expect(result).To(ContainSubstring(`ProxySet disablereuse=On retry=0`))
			Expect(result).To(ContainSubstring(`<Directory "/app/htdocs">`))
			Expect(result).To(ContainSubstring(`SetHandler proxy:fcgi://127.0.0.1:9000`))
			Expect(result).To(ContainSubstring(`RequestHeader unset Proxy early`))
			Expect(result).To(ContainSubstring(`IncludeOptional "/app/.httpd.conf.d/*.conf"`))
		})

		it("generates an httpd.conf and disables HTTPS redirection", func() {
			cfg := HttpdConfig{
				AppRoot:              "/app",
				ServerAdmin:          "test@example.org",
				WebDirectory:         "htdocs",
				FpmSocket:            "127.0.0.1:9000",
				DisableHTTPSRedirect: true,
			}

			err := ProcessTemplateToFile(HttpdConfTemplate, filepath.Join(f.Home, "httpd.conf"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "httpd.conf"))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(ContainSubstring(`RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301,NE]`))
		})

		it("generates an nginx.conf from the template", func() {
			cfg := NginxConfig{
				AppRoot:      "/app",
				WebDirectory: "public",
				FpmSocket:    "/tmp/php-fpm.socket",
			}

			err := ProcessTemplateToFile(NginxConfTemplate, filepath.Join(f.Home, "nginx.conf"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "nginx.conf"))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`root               /app/public;`))
			Expect(result).To(ContainSubstring(`server unix:/tmp/php-fpm.socket;`))
			Expect(result).To(ContainSubstring(`listen       {{env "PORT"}}  default_server;`))
			Expect(result).To(ContainSubstring(`map $http_x_forwarded_proto $redirect_to_https {`))
			Expect(result).To(ContainSubstring(`if ($redirect_to_https = "yes") {`))
			Expect(result).To(ContainSubstring(`return 301 https://$http_host$request_uri;`))
			Expect(string(result)).To(ContainSubstring(`include /app/.nginx.conf.d/*-server.conf`))
			Expect(string(result)).To(ContainSubstring(`include /app/.nginx.conf.d/*-http.conf`))
		})

		it("generates an nginx.conf and disables HTTPS redirection", func() {
			cfg := NginxConfig{
				AppRoot:              "/app",
				WebDirectory:         "public",
				FpmSocket:            "/tmp/php-fpm.socket",
				DisableHTTPSRedirect: true,
			}

			err := ProcessTemplateToFile(NginxConfTemplate, filepath.Join(f.Home, "nginx.conf"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "nginx.conf"))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(ContainSubstring(`map $http_x_forwarded_proto $redirect_to_https {`))
			Expect(result).ToNot(ContainSubstring(`if ($redirect_to_https = "yes") {`))
			Expect(result).ToNot(ContainSubstring(`return 301 https://$http_host$request_uri;`))
		})

		it("generates a php.ini from the template", func() {
			cfg := PhpIniConfig{
				AppRoot:      "/app",
				LibDirectory: "lib",
				PhpHome:      "/php/home",
				PhpAPI:       "20180101",
				Extensions: []string{
					"openssl",
					"mysql",
				},
				ZendExtensions: []string{
					"xdebug",
				},
			}

			err := ProcessTemplateToFile(PhpIniTemplate, filepath.Join(f.Home, "php.ini"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "php.ini"))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`include_path = "/php/home/lib/php:/app/lib"`))
			Expect(result).To(ContainSubstring(`extension_dir = "/php/home/lib/php/extensions/no-debug-non-zts-20180101"`))
			Expect(result).To(ContainSubstring(`extension = openssl.so`))
			Expect(result).To(ContainSubstring(`extension = mysql.so`))
			Expect(result).To(ContainSubstring(`zend_extension = xdebug.so`))
		})

		it("generates a php-fpm.conf from the template", func() {
			cfg := PhpFpmConfig{
				Include: "/php/home/.php-fpm.d/*.conf",
				Listen:  "127.0.0.1:9000",
			}

			err := ProcessTemplateToFile(PhpFpmConfTemplate, filepath.Join(f.Home, "php-fpm.conf"), cfg)
			Expect(err).ToNot(HaveOccurred())

			result, err := ioutil.ReadFile(filepath.Join(f.Home, "php-fpm.conf"))

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`include=/php/home/.php-fpm.d/*.conf`))
			Expect(result).To(ContainSubstring(`listen = 127.0.0.1:9000`))
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
			Expect(loaded).To(Equal(BuildpackYAML{
				Config{
					Version:             "",
					WebServer:           "php-server",
					WebDirectory:        "htdocs",
					LibDirectory:        "lib",
					Script:              "",
					ServerAdmin:         "admin@localhost",
					EnableHTTPSRedirect: true,
					Redis: Redis{
						SessionStoreServiceName: "redis-sessions",
					},
					Memcached: Memcached{
						SessionStoreServiceName: "memcached-sessions",
					},
				},
			}))
		})

		it("can load a version & web server", func() {
			yaml := "{'php': {'version': 1.0.0, 'webserver': 'httpd', 'serveradmin': 'admin@example.com', 'enable_https_redirect': false}}"
			test.WriteFile(t, filepath.Join(f.Detect.Application.Root, "buildpack.yml"), yaml)

			loaded, err := LoadBuildpackYAML(f.Detect.Application.Root)
			actual := BuildpackYAML{
				Config: Config{
					Version:             "1.0.0",
					WebServer:           "httpd",
					WebDirectory:        "htdocs",
					LibDirectory:        "lib",
					Script:              "",
					ServerAdmin:         "admin@example.com",
					EnableHTTPSRedirect: false,
					Redis: Redis{
						SessionStoreServiceName: "redis-sessions",
					},
					Memcached: Memcached{
						SessionStoreServiceName: "memcached-sessions",
					},
				},
			}

			Expect(err).To(Succeed())
			Expect(loaded).To(Equal(actual))
		})

		it("logs a warning against user-set buildpack.yml config", func() {
			yaml := `{'php':
			{
				'version': 1.0.0,
				'webserver': 'httpd',
				'serveradmin': 'admin@example.com',
				'libdirectory': 'some-libdir',
				'webdirectory': 'some-webdir',
				'script': 'some-script',
				'enable_https_redirect': true,
				'redis': {'session_store_service_name': 'redis-session-store-name'},
				'memcached': {'session_store_service_name': 'memcached-session-store-name'}
		}}`
			test.WriteFile(t, filepath.Join(f.Build.Application.Root, "buildpack.yml"), yaml)

			buf := bytes.NewBuffer(nil)

			logger := logger.Logger{Logger: bp.NewLogger(buf, buf)}
			Expect(WarnBuildpackYAML(logger, "1.2.3", f.Detect.Application.Root)).To(Succeed())
			Expect(buf.String()).To(ContainSubstring(`WARNING: Setting PHP configurations through buildpack.yml will be deprecated soon in buildpack v2.0.0.`))
			Expect(buf.String()).To(ContainSubstring("Buildpack.yml values will be replaced by environment variables in the next major version:"))
			Expect(buf.String()).To(ContainSubstring("php.version -> BP_PHP_VERSION"))
			Expect(buf.String()).To(ContainSubstring("php.webserver -> BP_PHP_SERVER"))
			Expect(buf.String()).To(ContainSubstring("php.serveradmin -> BP_PHP_SERVER_ADMIN"))
			Expect(buf.String()).To(ContainSubstring("php.libdirectory -> BP_PHP_LIB_DIR"))
			Expect(buf.String()).To(ContainSubstring("php.webdirectory -> BP_PHP_WEB_DIR"))
			Expect(buf.String()).To(ContainSubstring("php.script -> use a Procfile"))
			Expect(buf.String()).To(ContainSubstring("php.enable_https_redirect -> BP_PHP_ENABLE_HTTPS_REDIRECT"))
			Expect(buf.String()).To(ContainSubstring("php.redis.session_store_service_name -> use a service binding"))
			Expect(buf.String()).To(ContainSubstring("php.memcached.session_store_service_name -> use a service binding"))
		})

		when("the buildpack.yml is empty", func() {
			it("does not log a warning against user-set buildpack.yml config", func() {
				buf := bytes.NewBuffer(nil)
				logger := logger.Logger{Logger: bp.NewLogger(buf, buf)}
				Expect(WarnBuildpackYAML(logger, "1.2.3", f.Detect.Application.Root)).To(Succeed())
				Expect(buf.String()).NotTo(MatchRegexp(`WARNING: Setting the PHP configurations through buildpack.yml will be deprecated soon in buildpack v\d+.\d+.\d+.`))
			})
		})
	})

	when("checking for a web app", func() {
		it("defaults `php.webdir` to `htdocs`", func() {
			Expect(PickWebDir(BuildpackYAML{})).To(Equal("htdocs"))
		})

		it("loads `php.webdirectory` from `buildpack.yml`", func() {
			buildpackYAML := BuildpackYAML{
				Config: Config{
					WebDirectory: "public",
				},
			}

			Expect(PickWebDir(buildpackYAML)).To(Equal("public"))
		})

		it("finds a web app under `<webdir>/*.php`", func() {
			test.WriteFile(t, filepath.Join(f.Build.Application.Root, "htdocs", "index.php"), "")
			found, err := SearchForWebApp(f.Build.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeTrue())
		})

		it("doesn't find a web app under `<webdir>/*.php`", func() {
			found, err := SearchForWebApp(f.Build.Application.Root, "htdocs")
			Expect(err).To(Not(HaveOccurred()))
			Expect(found).To(BeFalse())
		})

	})
}
