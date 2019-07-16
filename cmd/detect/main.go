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

	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/php-cnb/php"

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

	if err := detectionContext.BuildPlan.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize Build Plan: %s\n", err)
		os.Exit(101)
	}

	code, err := runDetect(detectionContext)
	if err != nil {
		detectionContext.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func pickWebDir(buildpackYAML phpweb.BuildpackYAML) string {
	if buildpackYAML.Config.WebDirectory != "" {
		return buildpackYAML.Config.WebDirectory
	}

	return "htdocs"
}

func searchForWebApp(appRoot string, webdir string) (bool, error) {
	matchList, err := filepath.Glob(filepath.Join(appRoot, webdir, "*.php"))
	if err != nil {
		return false, err
	}

	if len(matchList) > 0 {
		return true, nil
	}
	return false, nil
}

func searchForScript(appRoot string, log logger.Logger) (bool, error) {
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

	plan, phpFound := context.BuildPlan[php.Dependency]
	if !phpFound {
		context.Logger.Body("PHP not listed in build plan, and is required")
		return context.Fail(), nil
	}
	version := phpweb.Version(buildpackYAML, context.Buildpack, plan)
	webDir := pickWebDir(buildpackYAML)

	webAppFound, err := searchForWebApp(context.Application.Root, webDir)
	if err != nil {
		return context.Fail(), err
	}

	if webAppFound {
		return context.Pass(buildplan.BuildPlan{
			php.Dependency: buildplan.Dependency{
				Metadata: buildplan.Metadata{
					"launch": true,
					"build":  true,
				},
				Version: version,
			},
			phpweb.WebDependency: buildplan.Dependency{},
			pickWebServer(buildpackYAML): buildplan.Dependency{
				Metadata: buildplan.Metadata{
					"launch": true,
				},
			},
		})
	}

	scriptFound, err := searchForScript(context.Application.Root, context.Logger)
	if err != nil {
		return context.Fail(), err
	}

	if scriptFound {
		return context.Pass(buildplan.BuildPlan{
			php.Dependency: buildplan.Dependency{
				Metadata: buildplan.Metadata{
					"launch": true,
					"build":  true,
				},
				Version: version,
			},
			phpweb.ScriptDependency: buildplan.Dependency{},
		})
	}

	return context.Fail(), nil
}

func pickWebServer(bpYaml phpweb.BuildpackYAML) string {
	webServer := httpd.Dependency
	if len(bpYaml.Config.WebServer) > 0 {
		webServer = bpYaml.Config.WebServer
	}
	return webServer
}
