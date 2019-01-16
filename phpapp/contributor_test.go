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

package phpapp

import (
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitContributor(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Contributor", testContributor, spec.Report(report.Terminal{}))
}

func testContributor(t *testing.T, when spec.G, it spec.S) {
	var f *test.BuildFactory

	it.Before(func() {
		f = test.NewBuildFactory(t)
	})

	it("returns true if build plan exists", func() {
		// f.AddDependency(Dependency, stubPHPFixture)
		f.AddBuildPlan(Dependency, buildplan.Dependency{})

		_, ok, err := NewContributor(f.Build)
		Expect(ok).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
	})

	it("returns false if build plan does not exist", func() {
		_, ok, err := NewContributor(f.Build)
		Expect(ok).To(BeFalse())
		Expect(err).NotTo(HaveOccurred())
	})
}
