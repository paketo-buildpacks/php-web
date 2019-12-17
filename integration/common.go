package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

var (
	phpDistURI, httpdURI, nginxURI, phpWebURI string
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() error {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return err
	}

	phpDistURI, err = dagger.GetLatestBuildpack("php-dist-cnb")
	if err != nil {
		return err
	}

	httpdURI, err = dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return err
	}

	nginxURI, err = dagger.GetLatestBuildpack("nginx-cnb")
	if err != nil {
		return err
	}

	phpWebURI, err = dagger.PackageBuildpack(bpRoot)
	if err != nil {
		return err
	}

	return nil
}

// CleanUpBps removes the packaged buildpacks
func CleanUpBps() {
	for _, bp := range []string{phpDistURI, httpdURI, nginxURI, phpWebURI} {
		dagger.DeleteBuildpack(bp)
	}
}

func PreparePhpApp(appName string, buildpacks []string, env map[string]string) (*dagger.App, error) {
	app, err := dagger.PackBuildWithEnv(filepath.Join("testdata", appName), env, buildpacks...)
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
