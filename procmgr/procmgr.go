package procmgr

import (
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"gopkg.in/yaml.v2"
)

// Procs is the list of process names and commands to run
type Procs struct {
	Processes map[string]Proc
}

// Proc is a single process to run
type Proc struct {
	Command string
	Args    []string
}

func ReadProcs(path string) (Procs, error) {
	procs := Procs{}

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Procs{
			Processes: map[string]Proc{},
		}, nil
	} else if err != nil {
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

func WriteProcs(path string, procs Procs) error {
	bytes, err := yaml.Marshal(procs)
	if err != nil {
		return err
	}
	return helper.WriteFile(path, 0644, string(bytes))
}

// AppendOrUpdateProcs appends or updates the given procs to the current proc.yml
func AppendOrUpdateProcs(path string, procs Procs) error {
	existingProcs, err := ReadProcs(path)
	if err != nil {
		return err
	}

	for name, proc := range procs.Processes {
		existingProcs.Processes[name] = proc
	}

	return WriteProcs(path, existingProcs)
}
