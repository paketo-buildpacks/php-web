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

package main

import (
	"testing"

	"github.com/cloudfoundry/php-web-cnb/procmgr"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitProcmgr(t *testing.T) {
	spec.Run(t, "Procmgr", testProcmgr, spec.Report(report.Terminal{}))
}

func testProcmgr(t *testing.T, _ spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	it("should run a proc", func() {
		err := runProcs(procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"proc1": {
					Command: "echo",
					Args:    []string{"'Hello World!"},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	it("should fail when running a proc that doesn't exist", func() {
		err := runProcs(procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"proc1": {
					Command: "idontexist",
					Args:    []string{},
				},
			},
		})
		Expect(err).To(HaveOccurred())
	})

	it("should run two procs", func() {
		err := runProcs(procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"proc1": {
					Command: "echo",
					Args:    []string{"'Hello World!"},
				},
				"proc2": {
					Command: "echo",
					Args:    []string{"'Good-bye World!"},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	it("should fail if proc exits non-zero", func() {
		err := runProcs(procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"proc1": {
					Command: "false",
					Args:    []string{""},
				},
			},
		})
		Expect(err).To(HaveOccurred())
	})

	it("should run two procs, where one is shorter", func() {
		err := runProcs(procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"sleep0.25": {
					Command: "sleep",
					Args:    []string{"0.25"},
				},
				"sleep1": {
					Command: "sleep",
					Args:    []string{"1"},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})
}
