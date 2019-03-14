package dagger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/libcfbuildpack/helper"
)

const (
	CFLINUXFS3 = "org.cloudfoundry.stacks.cflinuxfs3"
	BIONIC     = "io.buildpacks.stacks.bionic"
)

var downloadCache sync.Map

func init() {
	rand.Seed(time.Now().UnixNano())
	downloadCache = sync.Map{}
}

func PackageBuildpack() (string, error) {
	cmd := exec.Command("../scripts/package.sh")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	r := regexp.MustCompile("Buildpack packaged into: (.*)")
	bpDir := r.FindStringSubmatch(string(out))[1]
	return bpDir, nil
}

func GetLatestBuildpack(name string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/cloudfoundry/%s/releases/latest", name))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	release := struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if len(release.Assets) == 0 {
		return "", fmt.Errorf("there are no releases for %s", name)
	}

	contents, found := downloadCache.Load(name + release.TagName)
	if !found {
		buildpackResp, err := http.Get(release.Assets[0].BrowserDownloadURL)
		if err != nil {
			return "", err
		}
		defer buildpackResp.Body.Close()

		contents, err = ioutil.ReadAll(buildpackResp.Body)
		if err != nil {
			return "", err
		}

		downloadCache.Store(name+release.TagName, contents)
	}

	downloadFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer os.Remove(downloadFile.Name())

	_, err = io.Copy(downloadFile, bytes.NewReader(contents.([]byte)))
	if err != nil {
		return "", err
	}

	dest, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	return dest, helper.ExtractTarGz(downloadFile.Name(), dest, 0)
}

func PackBuild(appDir string, buildpacks ...string) (*App, error) {
	appImageName := randomString(16)
	buildStdout := &bytes.Buffer{}
	buildStderr := &bytes.Buffer{}

	cmd := exec.Command("pack", "build", appImageName, "--builder", "cfbuildpacks/cflinuxfs3-cnb-test-builder", "--clear-cache")
	for _, bp := range buildpacks {
		cmd.Args = append(cmd.Args, "--buildpack", bp)
	}
	cmd.Dir = appDir
	cmd.Stdout = io.MultiWriter(os.Stdout, buildStdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, buildStderr)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	app := &App{
		BuildStderr: buildStderr,
		BuildStdout: buildStdout,
		Env:         make(map[string]string),
		imageName:   appImageName,
		fixtureName: appDir,
	}
	return app, nil
}

type App struct {
	BuildStdout *bytes.Buffer
	BuildStderr *bytes.Buffer
	Env         map[string]string
	logProc     *exec.Cmd
	imageName   string
	containerId string
	port        string
	fixtureName string
	healthCheck HealthCheck
}

type HealthCheck struct {
	command  string
	interval string
	timeout  string
}

func (a *App) SetHealthCheck(command, interval, timeout string) {
	a.healthCheck = HealthCheck{
		command:  command,
		interval: interval,
		timeout:  timeout,
	}
}

func (a *App) Start() error {
	buf := &bytes.Buffer{}

	args := []string{"run", "-d", "-P"}
	if a.healthCheck.command != "" {
		args = append(args, "--health-cmd", a.healthCheck.command)
	}

	if a.healthCheck.interval != "" {
		args = append(args, "--health-interval", a.healthCheck.interval)
	}

	if a.healthCheck.timeout != "" {
		args = append(args, "--health-timeout", a.healthCheck.timeout)
	}

	envTemplate := "%s=%s"
	for k, v := range a.Env {
		envString := fmt.Sprintf(envTemplate, k, v)
		args = append(args, "-e", envString)
	}

	args = append(args, a.imageName)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	a.containerId = buf.String()[:12]

	ticker := time.NewTicker(1 * time.Second)
	timeOut := time.After(40 * time.Second)
docker:
	for {
		select {
		case <-ticker.C:
			status, err := exec.Command("docker", "inspect", "-f", "{{.State.Health.Status}}", a.containerId).Output()
			if err != nil {
				return err
			}

			if strings.TrimSpace(string(status)) == "unhealthy" {
				return fmt.Errorf("app failed to start : %s", a.fixtureName)
			}

			if strings.TrimSpace(string(status)) == "healthy" {
				break docker
			}
		case <-timeOut:
			return fmt.Errorf("timed out waiting for app : %s", a.fixtureName)
		}
	}

	cmd = exec.Command("docker", "container", "port", a.containerId)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	a.port = strings.TrimSpace(strings.Split(buf.String(), ":")[1])

	return nil
}

func (a *App) Destroy() error {
	if a.containerId == "" {
		return nil
	}

	cmd := exec.Command("docker", "stop", a.containerId)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("docker", "rm", a.containerId, "-f", "--volumes")
	if err := cmd.Run(); err != nil {
		return err
	}

	a.containerId = ""
	a.port = ""

	if a.imageName == "" {
		return nil
	}

	cmd = exec.Command("docker", "rmi", a.imageName, "-f")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("docker", "image", "prune", "-f")
	if err := cmd.Run(); err != nil {
		return err
	}

	a.imageName = ""

	return nil
}

func (a *App) Info() (cID string, imageID string, cacheID []string, e error) {
	volumes, err := getCacheVolumes()
	if err != nil {
		return "", "", []string{}, err
	}

	return a.containerId, a.imageName, volumes, nil
}

func (a *App) Logs() (string, error) {
	cmd := exec.Command("docker", "logs", a.containerId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (a *App) HTTPGet(path string) (string, map[string][]string, error) {
	resp, err := http.Get("http://localhost:" + a.port + path)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("received bad response from application")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	return string(body), resp.Header, nil
}

func getCacheVolumes() ([]string, error) {
	cmd := exec.Command("docker", "volume", "ls", "-q")
	output, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}

	outputArr := strings.Split(string(output), "\n")
	var finalVolumes []string
	for _, line := range outputArr {
		if strings.Contains(line, "pack-cache") {
			finalVolumes = append(finalVolumes, line)
		}
	}
	return outputArr, nil
}

func randomString(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}