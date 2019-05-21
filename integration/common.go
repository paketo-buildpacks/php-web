package integration

import (
	"github.com/cloudfoundry/dagger"
	"path/filepath"
)


func PrepareBuildpack() (phpWebBp, phpBp, httpdBp string, err error) {
	root, err := dagger.FindBPRoot()
	if err != nil {
		return "", "", "", err
	}

	phpWebBp, err = dagger.PackageBuildpack(root)
	if err != nil {
		return "", "", "", err
	}

	phpBp, err = dagger.GetLatestBuildpack("php-cnb")
	if err != nil {
		return "", "", "", err
	}

	httpdBp, err = dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return "", "", "", err
	}

	return phpWebBp, phpBp, httpdBp, nil
}

func PreparePhpApp(appName string, buildpacks ...string) (*dagger.App, error) {
	app, err := dagger.PackBuild(filepath.Join("testdata", appName), buildpacks...)
	if err != nil {
		return &dagger.App{}, nil
	}

	app.SetHealthCheck("", "3s", "1s")

	return app, nil
}
