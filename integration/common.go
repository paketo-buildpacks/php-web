package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/dagger"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var (
	phpDistURI        string
	phpDistOfflineURI string
	httpdURI          string
	httpdOfflineURI   string
	nginxURI          string
	nginxOfflineURI   string
	phpWebURI         string
	phpWebOfflineURI  string
	version           string
	buildpackInfo     struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() error {
	var config struct {
		Httpd   string `json:"httpd"`
		Nginx   string `json:"nginx"`
		PhpDist string `json:"php-dist"`
	}

	file, err := os.Open("../integration.json")
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())

	bpRoot, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	version, err = GetGitVersion()
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	phpDistURI, err = buildpackStore.Get.Execute(config.PhpDist)
	Expect(err).ToNot(HaveOccurred())

	phpDistRepo, err := dagger.GetLatestUnpackagedBuildpack("php-dist-cnb")
	Expect(err).ToNot(HaveOccurred())

	phpDistOfflineURI, err = Package(phpDistRepo, "1.2.3", true)
	Expect(err).ToNot(HaveOccurred())

	httpdURI, err = buildpackStore.Get.Execute(config.Httpd)
	Expect(err).ToNot(HaveOccurred())

	httpdOfflineURI, err = buildpackStore.Get.WithOfflineDependencies().Execute(config.Httpd)
	Expect(err).ToNot(HaveOccurred())

	nginxURI, err = buildpackStore.Get.Execute(config.Nginx)
	Expect(err).ToNot(HaveOccurred())

	nginxOfflineURI, err = buildpackStore.Get.WithOfflineDependencies().Execute(config.Nginx)
	Expect(err).ToNot(HaveOccurred())

	phpWebURI, err = Package(bpRoot, version, false)
	Expect(err).ToNot(HaveOccurred())

	phpWebOfflineURI, err = Package(bpRoot, version, true)
	Expect(err).ToNot(HaveOccurred())

	return nil
}

// CleanUpBps removes the packaged buildpacks
func CleanUpBps() {
	for _, bp := range []string{phpDistURI, phpDistOfflineURI, phpWebURI, phpWebOfflineURI} {
		Expect(dagger.DeleteBuildpack(bp)).To(Succeed())
	}
}

func PreparePhpApp(appName string, buildpacks []string, env map[string]string) (*dagger.App, error) {
	app, err := dagger.NewPack(
		filepath.Join("testdata", appName),
		dagger.RandomImage(),
		dagger.SetEnv(env),
		dagger.SetBuildpacks(buildpacks...),
		dagger.SetVerbose(),
	).Build()
	if err != nil {
		return nil, err
	}

	app.SetHealthCheck("", "3s", "1s")
	if env == nil {
		env = make(map[string]string)
	}
	env["PORT"] = "8080"
	app.Env = env

	return app, nil
}

func PushSimpleApp(name string, buildpacks []string, script bool) (*dagger.App, error) {
	app, err := PreparePhpApp(name, buildpacks, nil)
	if err != nil {
		return app, err
	}

	if script {
		app.SetHealthCheck("true", "3s", "1s")
	}

	err = app.Start()
	if err != nil {
		_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
		if err != nil {
			return app, err
		}

		containerID, imageName, volumeIDs, err := app.Info()
		if err != nil {
			return app, err
		}

		fmt.Printf("ContainerID: %s\nImage Name: %s\nAll leftover cached volumes: %v\n", containerID, imageName, volumeIDs)

		containerLogs, err := app.Logs()
		if err != nil {
			return app, err
		}

		fmt.Printf("Container Logs:\n %s\n", containerLogs)
		return app, err
	}

	return app, nil
}

func Package(root, version string, cached bool) (string, error) {
	var cmd *exec.Cmd

	dir, err := filepath.Abs("./..")
	if err != nil {
		return "", err
	}

	bpPath := filepath.Join(root, "artifact")
	if cached {
		cmd = exec.Command(filepath.Join(dir, ".bin", "packager"), "--archive", "--version", version, fmt.Sprintf("%s-cached", bpPath))
	} else {
		cmd = exec.Command(filepath.Join(dir, ".bin", "packager"), "--archive", "--uncached", "--version", version, bpPath)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PACKAGE_DIR=%s", bpPath))
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	if cached {
		return fmt.Sprintf("%s-cached.tgz", bpPath), nil
	}

	return fmt.Sprintf("%s.tgz", bpPath), nil
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	revListOut := bytes.NewBuffer(nil)

	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"rev-list", "--tags", "--max-count=1"},
		Stdout: revListOut,
	})
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	err = gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--tags", strings.TrimSpace(revListOut.String())},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}

// later todo: move this matcher to occam
func BeAvailableAndReady() types.GomegaMatcher {
	return &BeAvailableAndReadyMatcher{
		Docker: occam.NewDocker(),
	}
}

type BeAvailableAndReadyMatcher struct {
	Docker occam.Docker
}

func (*BeAvailableAndReadyMatcher) Match(actual interface{}) (bool, error) {
	container, ok := actual.(occam.Container)
	if !ok {
		return false, fmt.Errorf("BeAvailableMatcher expects an occam.Container, received %T", actual)
	}

	response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort()))
	if err != nil {
		return false, nil
	}

	if response.StatusCode != http.StatusOK {
		return false, nil
	}

	defer response.Body.Close()

	return true, nil
}

func (m *BeAvailableAndReadyMatcher) FailureMessage(actual interface{}) string {
	container := actual.(occam.Container)
	message := fmt.Sprintf("Expected\n\tdocker container id: %s\nto be available.", container.ID)

	if logs, _ := m.Docker.Container.Logs.Execute(container.ID); logs != nil {
		message = fmt.Sprintf("%s\n\nContainer logs:\n\n%s", message, logs)
	}

	return message
}

func (m *BeAvailableAndReadyMatcher) NegatedFailureMessage(actual interface{}) string {
	container := actual.(occam.Container)
	message := fmt.Sprintf("Expected\n\tdocker container id: %s\nnot to be available.", container.ID)

	if logs, _ := m.Docker.Container.Logs.Execute(container.ID); logs != nil {
		message = fmt.Sprintf("%s\n\nContainer logs:\n\n%s", message, logs)
	}

	return message
}
