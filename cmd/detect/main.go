/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"

	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/php-dist-cnb/php"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-web-cnb/phpweb"
)

func main() {
	detectionContext, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to run detection: %s", err)
		os.Exit(101)
	}

	code, err := runDetect(detectionContext)
	if err != nil {
		detectionContext.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func searchForAnyPHPFiles(appRoot string, log logger.Logger) (bool, error) {
	found := false

	err := filepath.Walk(appRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Info("failure accessing a path %q: %v\n", path, err)
			return filepath.SkipDir
		}

		if found {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".php") {
			found = true
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return found, nil
}

func runDetect(context detect.Detect) (int, error) {
	buildpackYAML, err := phpweb.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	webDir := phpweb.PickWebDir(buildpackYAML)
	version := phpweb.Version(context.Buildpack)
	isWebApp, err := phpweb.SearchForWebApp(context.Application.Root, webDir)
	if err != nil {
		return context.Fail(), err
	}

	hasAnyPHPFiles, err := searchForAnyPHPFiles(context.Application.Root, context.Logger)
	if err != nil {
		return context.Fail(), err
	}

	if !(isWebApp || hasAnyPHPFiles) {
		return context.Fail(), nil
	}

	plan := buildplan.Plan{
		Provides: []buildplan.Provided{
			{
				Name: phpweb.Dependency,
			},
		},
		Requires: []buildplan.Required{
			requiredPHP(version),
			{
				Name: phpweb.Dependency,
			},
		},
	}

	if isWebApp {
		webServer := pickWebServer(buildpackYAML)
		plan.Requires = append(plan.Requires, buildplan.Required{
			Name:     webServer,
			Metadata: buildplan.Metadata{"launch": true},
		})

		if webServer == phpweb.PhpWebServer {
			plan.Provides = append(plan.Provides, buildplan.Provided{
				Name: phpweb.PhpWebServer,
			})
		}
	}

	return context.Pass(plan)
}

func requiredPHP(version string) buildplan.Required {
	return buildplan.Required{
		Name:    php.Dependency,
		Version: version,
		Metadata: buildplan.Metadata{
			"launch":                    true,
			"build":                     true,
			buildpackplan.VersionSource: php.DefaultVersionsSource,
		},
	}
}

func pickWebServer(bpYaml phpweb.BuildpackYAML) string {
	webServer := httpd.Dependency
	if len(bpYaml.Config.WebServer) > 0 {
		webServer = bpYaml.Config.WebServer
	}
	return webServer
}
