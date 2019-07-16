package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() ([]string, error) {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return []string{}, err
	}

	phpBp, err := dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return []string{}, err
	}

	httpdBp, err := dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return []string{}, err
	}

	nginxBp, err := dagger.GetLatestBuildpack("nginx-cnb")
	if err != nil {
		return []string{}, err
	}

	phpWebBp, err := dagger.PackageBuildpack(bpRoot)
	if err != nil {
		return []string{}, err
	}

	return []string{phpBp, httpdBp, nginxBp, phpWebBp}, nil
}

// MakeBuildEnv creates a build environment map
func MakeBuildEnv(debug bool) map[string]string {
	env := make(map[string]string)
	if debug {
		env["BP_DEBUG"] = "true"
	}

	return env
}

func PreparePhpApp(appName string, buildpacks []string, debug bool) (*dagger.App, error) {
	app, err := dagger.PackBuildWithEnv(filepath.Join("testdata", appName), MakeBuildEnv(debug), buildpacks...)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}

func PushSimpleApp(name string, buildpacks []string, script bool) (*dagger.App, error) {
	app, err := PreparePhpApp(name, buildpacks, false)
	if err != nil {
		return app, err
	}

	if script {
		app.SetHealthCheck("true", "3s", "1s")
	}

	err = app.Start()

	if err != nil {
		_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
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
