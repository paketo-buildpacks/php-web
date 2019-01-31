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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitProcmgr(t *testing.T) {
	spec.Run(t, "Procmgr", testProcmgr, spec.Report(report.Terminal{}))
}

func testProcmgr(t *testing.T, when spec.G, it spec.S) {
	var tmp string

	it.Before(func() {
		RegisterTestingT(t)

		var err error
		tmp, err = ioutil.TempDir("", "procmgr")
		Expect(err).ToNot(HaveOccurred())
	})

	it("should load some procs", func() {
		procs := `{"processes": {"echo1": {"command": "echo", "args": "'Hello World!'"}, "echo2": {"command": "echo", "args": "'Good-bye World!'"}}}`
		procsFile := filepath.Join(tmp, "procs.yml")
		err := writeYaml(procsFile, procs)

		Expect(err).ToNot(HaveOccurred())
		Expect(procsFile).To(BeARegularFile())

		list, err := readProcs(procsFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(list.Processes)).To(Equal(2))
		Expect(list.Processes["echo1"]).To(Equal(Proc{"echo", "'Hello World!'"}))
	})

	it("should fail on bad yaml", func() {
		procs := `Not actuall YAML`
		procsFile := filepath.Join(tmp, "procs.yml")
		err := writeYaml(procsFile, procs)

		Expect(err).ToNot(HaveOccurred())
		Expect(procsFile).To(BeARegularFile())

		_, err = readProcs(procsFile)
		Expect(err).To(HaveOccurred())
	})

	it("should if file does not exist", func() {
		procsFile := filepath.Join(tmp, "procs.yml")

		Expect(procsFile).ToNot(BeARegularFile())

		_, err := readProcs(procsFile)
		Expect(err).To(HaveOccurred())
	})

	it("should run a proc", func() {
		err := runProcs(Procs{
			Processes: map[string]Proc{
				"proc1": Proc{
					Command: "echo",
					Args:    "'Hello World!",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	it("should run two procs", func() {
		err := runProcs(Procs{
			Processes: map[string]Proc{
				"proc1": Proc{
					Command: "echo",
					Args:    "'Hello World!",
				},
				"proc2": Proc{
					Command: "echo",
					Args:    "'Good-bye World!",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	it("should fail if proc exits non-zero", func() {
		err := runProcs(Procs{
			Processes: map[string]Proc{
				"proc1": Proc{
					Command: "false",
					Args:    "",
				},
			},
		})
		Expect(err).To(HaveOccurred())
	})

	it("should run two procs, where one is shorter", func() {
		err := runProcs(Procs{
			Processes: map[string]Proc{
				"sleep0.25": Proc{
					Command: "sleep",
					Args:    "0.25",
				},
				"sleep1": Proc{
					Command: "sleep",
					Args:    "1",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})
}

func writeYaml(path string, format string, args ...interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, []byte(fmt.Sprintf(format, args...)), 0644); err != nil {
		return err
	}

	return nil
}
