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

package httpd

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitHttpd(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Httpd", testHttpd, spec.Report(report.Terminal{}))
}

func testHttpd(t *testing.T, when spec.G, it spec.S) {
	it("generates a config from the template", func() {
		t := Template{
			ServerAdmin:  "test@example.org",
			WebDirectory: "htdocs",
			FpmSocket:    "127.0.0.1:9000",
		}

		cfg, err := t.Populate()

		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).To(ContainSubstring(`ServerRoot "${SERVER_ROOT}"`))
		Expect(cfg).To(ContainSubstring(`ServerAdmin "test@example.org"`))
		Expect(cfg).To(ContainSubstring(`DocumentRoot "${HOME}/htdocs"`))
		Expect(cfg).To(ContainSubstring(`<Directory "${HOME}/htdocs">`))
		Expect(cfg).To(ContainSubstring(`<Files ".ht*">`))
		Expect(cfg).To(ContainSubstring(`ErrorLog "/proc/self/fd/2"`))
		Expect(cfg).To(ContainSubstring(`CustomLog "/proc/self/fd/1" extended`))
		Expect(cfg).To(ContainSubstring(`RemoteIpHeader x-forwarded-for`))
		Expect(cfg).To(ContainSubstring(`RemoteIpInternalProxy 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16`))
		Expect(cfg).To(ContainSubstring(`SetEnvIf x-forwarded-proto https HTTPS=on`))
		Expect(cfg).To(ContainSubstring(`Define fcgi-listener fcgi://127.0.0.1:9000${HOME}/htdocs`))
		Expect(cfg).To(ContainSubstring(`<Proxy "${fcgi-listener}">`))
		Expect(cfg).To(ContainSubstring(`ProxySet disablereuse=On retry=0`))
		Expect(cfg).To(ContainSubstring(`<Directory "${HOME}/htdocs">`))
		Expect(cfg).To(ContainSubstring(`SetHandler proxy:${fcgi-listener}`))
		Expect(cfg).To(ContainSubstring(`RequestHeader unset Proxy early`))
	})
}
