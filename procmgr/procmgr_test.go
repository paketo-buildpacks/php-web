package procmgr

import (
	"github.com/cloudfoundry/libcfbuildpack/helper"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestUnitProcmgrLib(t *testing.T) {
	spec.Run(t, "ProcmgrLib", testProcmgrLib, spec.Report(report.Terminal{}))
}

func testProcmgrLib(t *testing.T, when spec.G, it spec.S) {
	var tmp string

	it.Before(func() {
		RegisterTestingT(t)

		var err error
		tmp, err = ioutil.TempDir("", "procmgr")
		Expect(err).ToNot(HaveOccurred())
	})

	it("should load some procs", func() {
		procs := `{"processes": {"echo1": {"command": "echo", "args": ["'Hello World!'"]}, "echo2": {"command": "echo", "args": ["'Good-bye World!'"]}}}`
		procsFile := filepath.Join(tmp, "procs.yml")
		Expect(helper.WriteFile(procsFile, os.ModePerm, procs)).To(Succeed())

		list, err := ReadProcs(procsFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(list.Processes)).To(Equal(2))
		Expect(list.Processes["echo1"]).To(Equal(Proc{"echo", []string{"'Hello World!'"}}))
	})

	it("should fail on bad yaml", func() {
		procs := `Not actuall YAML`
		procsFile := filepath.Join(tmp, "procs.yml")
		Expect(helper.WriteFile(procsFile, os.ModePerm, procs)).To(Succeed())

		_, err := ReadProcs(procsFile)
		Expect(err).To(HaveOccurred())
	})

	it("should if file does not exist", func() {
		procsFile := filepath.Join(tmp, "procs.yml")

		Expect(procsFile).ToNot(BeARegularFile())

		_, err := ReadProcs(procsFile)
		Expect(err).To(HaveOccurred())
	})

	it("should write some procs", func() {
		path := filepath.Join(tmp, "procs.yml")

		procs := Procs{
			Processes: map[string]Proc{
				"proc1": {
					Command: "echo",
					Args:    []string{"'Hello World!"},
				},
				"proc2": {
					Command: "echo",
					Args:    []string{"'Good-bye World!"},
				},
			},
		}

		Expect(WriteProcs(path, procs)).To(Succeed())
		Expect(path).To(BeARegularFile())

		file, err := os.Open(path)
		Expect(err).NotTo(HaveOccurred())
		defer file.Close()

		buf, err := ioutil.ReadAll(file)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(buf)).To(ContainSubstring(`Hello World!`))
		Expect(string(buf)).To(ContainSubstring(`Good-bye World!`))
	})
}
