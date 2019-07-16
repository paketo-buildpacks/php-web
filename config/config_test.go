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
	"io/ioutil"
	"path/filepath"
	"testing"

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
		Expect(result).To(ContainSubstring(`<Files ".ht*">`))
		Expect(result).To(ContainSubstring(`ErrorLog "/proc/self/fd/2"`))
		Expect(result).To(ContainSubstring(`CustomLog "/proc/self/fd/1" extended`))
		Expect(result).To(ContainSubstring(`RemoteIpHeader x-forwarded-for`))
		Expect(result).To(ContainSubstring(`RemoteIpInternalProxy 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16`))
		Expect(result).To(ContainSubstring(`SetEnvIf x-forwarded-proto https HTTPS=on`))
		Expect(result).To(ContainSubstring(`Define fcgi-listener fcgi://127.0.0.1:9000/app/htdocs`))
		Expect(result).To(ContainSubstring(`<Proxy "${fcgi-listener}">`))
		Expect(result).To(ContainSubstring(`ProxySet disablereuse=On retry=0`))
		Expect(result).To(ContainSubstring(`<Directory "/app/htdocs">`))
		Expect(result).To(ContainSubstring(`SetHandler proxy:fcgi://127.0.0.1:9000`))
		Expect(result).To(ContainSubstring(`RequestHeader unset Proxy early`))
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
		Expect(result).To(ContainSubstring(`listen       {{env "PORT"}};`))
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
}
