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
	"os/exec"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "USAGE:")
		fmt.Fprintln(os.Stderr, "    procmgr <path-to-proc-file>")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	procs, err := readProcs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading/parsing procs file:", err)
		os.Exit(2)
	}

	runProcs(procs)
}

// Procs is the list of process names and commands to run
type Procs struct {
	Processes map[string]Proc
}

// Proc is a single process to run
type Proc struct {
	Command string
	Args    string
}

func readProcs(path string) (Procs, error) {
	procs := Procs{}

	file, err := os.Open(path)
	if err != nil {
		return Procs{}, err
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return Procs{}, err
	}

	err = yaml.UnmarshalStrict(contents, &procs)
	if err != nil {
		return Procs{}, err
	}

	return procs, nil
}

type procMsg struct {
	ProcName string
	Cmd      *exec.Cmd
	Err      error
}

func runProcs(procs Procs) error {
	msgs := make(chan procMsg)

	for procName, proc := range procs.Processes {
		go runProc(procName, proc, msgs)
	}

	select {
	case msg := <-msgs:
		fmt.Fprintln(os.Stderr, "process", msg.ProcName, "exited, status:", msg.Cmd.ProcessState)
		return msg.Err
	}
}

func runProc(procName string, proc Proc, msgs chan procMsg) {
	cmd := exec.Command(proc.Command, proc.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		msgs <- procMsg{procName, cmd, err}
	}

	err = cmd.Wait()
	if err != nil {
		msgs <- procMsg{procName, cmd, err}
	}

	msgs <- procMsg{procName, cmd, nil}
}
