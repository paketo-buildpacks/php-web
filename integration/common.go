package integration

import (
	"github.com/cloudfoundry/dagger"
	"path/filepath"
)

func PreparePhpApp (appName string) (*dagger.App, error) {
	phpWebBp, err := dagger.PackageBuildpack()
	if err != nil {
		return &dagger.App{}, nil
	}

	phpBp, err := dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return &dagger.App{}, nil
	}

	httpdBp, err := dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return &dagger.App{}, nil
	}

	app, err := dagger.PackBuild(filepath.Join("fixtures", appName), phpBp, httpdBp, phpWebBp)
	if err != nil {
		return &dagger.App{}, nil
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}