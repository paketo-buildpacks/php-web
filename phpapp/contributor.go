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

package phpapp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-cnb/php"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	application application.Application
	layers      layers.Layers
	logger      logger.Logger
	isWebApp    bool
	isScript    bool
	webserver   string
	webdir      string
	script      string
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-script" or "php-web" dependency, otherwise false.
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	buildpackYAML, err := php.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	_, isWebApp := context.BuildPlan[WebDependency]
	_, isScript := context.BuildPlan[ScriptDependency]

	contributor := Contributor{
		application: context.Application,
		layers:      context.Layers,
		logger:      context.Logger,
		isWebApp:    isWebApp,
		isScript:    isScript,
		webserver:   buildpackYAML.Config.WebServer,
		webdir:      buildpackYAML.Config.WebDirectory,
		script:      buildpackYAML.Config.Script,
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	if c.isWebApp {
		c.logger.FirstLine("Configuring PHP Web Application")

		if len(c.webdir) == 0 {
			c.webdir = "htdocs"
		}
		c.logger.SubsequentLine("Using web directory: %s", c.webdir)

		if len(c.webserver) == 0 {
			c.webserver = PhpWebServer
		}

		if strings.ToLower(c.webserver) == PhpWebServer {
			c.logger.SubsequentLine("Using PHP built-in server")
			webdir := filepath.Join(c.application.Root, c.webdir)
			command := fmt.Sprintf("php -S 0.0.0.0:8080 -t %s", webdir)

			return c.layers.WriteMetadata(layers.Metadata{
				Processes: []layers.Process{
					{"web", command},
					{"task", command},
				},
			})
		}

		if strings.ToLower(c.webserver) == ApacheHttpd {
			// TODO: write out httpd.conf to c.application.Root
			c.logger.SubsequentLine("Using Apache Web Server")
		}

		if strings.ToLower(c.webserver) == Nginx {
			// TODO: write out nginx.conf to c.application.Root
			c.logger.SubsequentLine("Using Nginx")
		}
	}

	if c.isScript {
		c.logger.FirstLine("Configuring PHP Script")

		if len(c.script) == 0 {
			c.script = "app.php"
		}
		scriptPath := filepath.Join(c.application.Root, c.script)

		scriptExists, err := helper.FileExists(scriptPath)
		if err != nil {
			return err
		}

		if !scriptExists {
			c.logger.Info("WARNING: `%s` start script not found. App will not start unless you specify a custom start command.", c.script)
		}

		command := fmt.Sprintf("php %s", scriptPath)

		return c.layers.WriteMetadata(layers.Metadata{
			Processes: []layers.Process{
				{"web", command},
				{"task", command},
			},
		})
	}

	c.logger.Info("WARNING: Did not detect either a web app or a PHP script to run. App will not start unless you specify a custom start command.")
	return nil
}
