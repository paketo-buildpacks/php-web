package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/dagger"
	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/packit/pexec"

	. "github.com/onsi/gomega"
)

var (
	phpDistURI              string
	phpDistOfflineURI       string
	httpdURI					      string
	httpdOfflineURI					string
	nginxURI					      string
	nginxOfflineURI					string
	phpWebURI					      string
	phpWebOfflineURI	      string
	version                 string
	buildpackInfo           struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() error {
	bpRoot, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	version, err = GetGitVersion()
	Expect(err).NotTo(HaveOccurred())

	// Later todo: These buildpack urls redirect from the old cf cnb urls.
	// When rewriting with packit, change them.
	phpDistURI, err = dagger.GetLatestBuildpack("php-dist-cnb")
	if err != nil {
		return err
	}

	phpDistRepo, err := dagger.GetLatestUnpackagedBuildpack("php-dist-cnb")
	Expect(err).ToNot(HaveOccurred())

	phpDistOfflineURI, _, err = dagger.PackageCachedBuildpack(phpDistRepo)
	Expect(err).ToNot(HaveOccurred())

	httpdURI, err = dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return err
	}

	nginxURI, err = dagger.GetLatestBuildpack("nginx-cnb")
	if err != nil {
		return err
	}

	nginxRepo, err := dagger.GetLatestUnpackagedBuildpack("nginx-cnb")
	Expect(err).ToNot(HaveOccurred())

	nginxOfflineURI, err = Package(nginxRepo, bpRoot, "1.2.3", true)
	Expect(err).ToNot(HaveOccurred())

	httpdRepo, err := dagger.GetLatestUnpackagedBuildpack("httpd-cnb")
	Expect(err).ToNot(HaveOccurred())

	httpdOfflineURI, err = Package(httpdRepo, bpRoot, "1.2.3", true)
	Expect(err).ToNot(HaveOccurred())

	phpWebURI, err = Package(bpRoot, bpRoot, version, false)
	Expect(err).ToNot(HaveOccurred())

	phpWebOfflineURI, err = Package(bpRoot, bpRoot, version, true)
	Expect(err).ToNot(HaveOccurred())

	return nil
}

// CleanUpBps removes the packaged buildpacks
func CleanUpBps() {
	for _, bp := range []string{phpDistURI, phpDistOfflineURI, httpdURI, httpdOfflineURI, nginxURI, nginxOfflineURI, phpWebURI, phpWebOfflineURI} {
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

func Package(root, packagerRoot, version string, cached bool) (string, error) {
	var cmd *exec.Cmd

	bpPath := filepath.Join(root, "artifact")
	if cached {
		cmd = exec.Command(".bin/packager", "--archive", "--version", version, fmt.Sprintf("%s-cached", bpPath))
	} else {
		cmd = exec.Command(".bin/packager", "--archive", "--uncached", "--version", version, bpPath)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PACKAGE_DIR=%s", bpPath))
	cmd.Dir = packagerRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if cached {
		return fmt.Sprintf("%s-cached.tgz", bpPath), err
	}

	return fmt.Sprintf("%s.tgz", bpPath), err
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
