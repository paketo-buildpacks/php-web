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

package phpweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	application   application.Application
	layers        layers.Layers
	logger        logger.Logger
	buildpackYAML config.BuildpackYAML
	procmgr       string
	metadata      Metadata
	features      []Feature
}

func generateRandomHash() [32]byte {
	randBuf := make([]byte, 512)
	rand.Read(randBuf)
	return sha256.Sum256(randBuf)
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-script" or "php-web" dependency, otherwise false.
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	shouldContribute := context.Plans.Has(Dependency)
	if !shouldContribute {
		return Contributor{}, false, nil
	}

	buildpackYAML, err := config.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	randomHash := generateRandomHash()

	contributor := Contributor{
		application:   context.Application,
		layers:        context.Layers,
		logger:        context.Logger,
		buildpackYAML: buildpackYAML,
		metadata:      Metadata{"PHP Web", hex.EncodeToString(randomHash[:])},
		features: []Feature{
			features.NewProcMgrFeature(filepath.Join(context.Buildpack.Root, "bin", "procmgr"), buildpackYAML),
			features.NewPhpWebServerFeature(context.Application, buildpackYAML),
			features.NewHttpdFeature(context.Application, buildpackYAML),
			features.NewNginxFeature(context.Application, buildpackYAML),
			features.NewPhpFpmFeature(context.Application, buildpackYAML),
			features.NewRedisFeature(context.Application, context.Services, buildpackYAML.Config.Redis.SessionStoreServiceName),
		},
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	l := c.layers.Layer(Dependency)

	webDir := PickWebDir(c.buildpackYAML)
	isWebApp, err := SearchForWebApp(c.application.Root, webDir)
	if err != nil {
		return err
	}

	if isWebApp {
		c.logger.Header("Configuring PHP Web Application")
		return l.Contribute(c.metadata, c.contributeWebApp, c.flags()...)
	}

	c.logger.Header("Configuring PHP Script")
	return l.Contribute(c.metadata, c.contributeScript, c.flags()...)
}

func (c Contributor) contributeWebApp(layer layers.Layer) error {
	if err := c.initPhp(layer); err != nil {
		return err
	}

	c.logger.Debug("Build Pack YAML: %v", c.buildpackYAML)

	// install features
	// TODO: test??
	//     - make sure feature is installed
	//     - check `phpinfo()` to confirm it's enabled and configured
	for _, feature := range c.features {
		if feature.IsNeeded() {
			c.logger.Body("Using feature -- %s", feature.Name())
			err := feature.EnableFeature(c.layers, layer)
			if err != nil {
				c.logger.BodyError("Failed %s", err)
				return err
			}
		} else {
			c.logger.Debug("Skipping feature -- %s", feature.Name())
		}
	}

	c.logger.BodyWarning("WARNING: Did not install requested web server: %s. Requested server is unavailable. Start command is not being provided.", c.buildpackYAML.Config.WebServer)
	return nil
}

func (c Contributor) contributeScript(layer layers.Layer) error {
	if err := c.initPhp(layer); err != nil {
		return err
	}

	if c.buildpackYAML.Config.Script == "" {
		for _, possible := range DefaultCliScripts {
			exists, err := helper.FileExists(filepath.Join(c.application.Root, possible))
			if err != nil {
				c.logger.BodyError(err.Error())
				// skip and continue
			}

			if exists {
				c.buildpackYAML.Config.Script = possible
				break
			}
		}

		if c.buildpackYAML.Config.Script == "" {
			c.buildpackYAML.Config.Script = "app.php"
			c.logger.BodyWarning("Buildpack could not find a file to execute. Either set php.script in buildpack.yml or include one of these files [%s]", strings.Join(DefaultCliScripts, ", "))
		}
	}

	scriptPath := filepath.Join(c.application.Root, c.buildpackYAML.Config.Script)
	command := fmt.Sprintf("php %s", scriptPath)

	return c.layers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{Type: "web", Command: command, Direct: false},
			{Type: "task", Command: command, Direct: false},
		},
	})
}

func (c Contributor) writePhpIni(layer layers.Layer) error {
	phpIniCfg := config.PhpIniConfig{
		AppRoot:      c.application.Root,
		LibDirectory: c.buildpackYAML.Config.LibDirectory,
		PhpHome:      os.Getenv("PHP_HOME"),
		PhpAPI:       os.Getenv("PHP_API"),
	}
	phpIniPath := filepath.Join(layer.Root, "etc", "php.ini")
	return config.ProcessTemplateToFile(config.PhpIniTemplate, phpIniPath, phpIniCfg)
}

func (c Contributor) initPhp(layer layers.Layer) error {
	if err := c.writePhpIni(layer); err != nil {
		return err
	}

	if err := layer.OverrideSharedEnv("PHPRC", filepath.Join(layer.Root, "etc")); err != nil {
		return err
	}

	return layer.OverrideSharedEnv("PHP_INI_SCAN_DIR", filepath.Join(c.application.Root, ".php.ini.d"))
}

func (c Contributor) flags() []layers.Flag {
	return []layers.Flag{
		layers.Launch,
	}
}
