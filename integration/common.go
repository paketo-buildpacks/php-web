package integration

import (
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() ([]string, error) {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return []string{}, err
	}

	phpWebBp, err := dagger.PackageBuildpack(bpRoot)
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

	return []string{phpBp, httpdBp, phpWebBp}, nil
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
