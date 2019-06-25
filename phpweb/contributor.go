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

	"github.com/cloudfoundry/php-web-cnb/procmgr"

	"github.com/cloudfoundry/php-web-cnb/config"

	"github.com/buildpack/libbuildpack/application"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-cnb/php"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	application   application.Application
	layers        layers.Layers
	logger        logger.Logger
	buildpackYAML BuildpackYAML
	phpDep        buildplan.Dependency
	isWebApp      bool
	isScript      bool
	procmgr       string
	metadata      Metadata
}

func generateRandomHash() [32]byte {
	randBuf := make([]byte, 512)
	rand.Read(randBuf)
	return sha256.Sum256(randBuf)
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-script" or "php-web" dependency, otherwise false.
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	_, isWebApp := context.BuildPlan[WebDependency]
	_, isScript := context.BuildPlan[ScriptDependency]
	phpDep, _ := context.BuildPlan[php.Dependency]

	randomHash := generateRandomHash()

	contributor := Contributor{
		application:   context.Application,
		layers:        context.Layers,
		logger:        context.Logger,
		buildpackYAML: buildpackYAML,
		phpDep:        phpDep,
		isWebApp:      isWebApp,
		isScript:      isScript,
		procmgr:       filepath.Join(context.Buildpack.Root, "bin", "procmgr"),
		metadata:      Metadata{"PHP Web", hex.EncodeToString(randomHash[:])},
	}

	return contributor, true, nil
}

func (c Contributor) writePhpIni(layer layers.Layer) error {
	phpIniCfg := config.PhpIniConfig{
		AppRoot:      c.application.Root,
		LibDirectory: c.buildpackYAML.Config.LibDirectory,
		PhpHome:      os.Getenv("PHP_HOME"),
		PhpAPI:       os.Getenv("PHP_API"),
	}
	phpIniPath := filepath.Join(layer.Root, "etc", "php.ini")
	if err := config.ProcessTemplateToFile(config.PhpIniTemplate, phpIniPath, phpIniCfg); err != nil {
		return err
	}
	return nil
}

func (c Contributor) writeHttpdConf(layer layers.Layer) error {
	httpdCfg := config.HttpdConfig{
		ServerAdmin:  "admin@localhost", //TODO: pull from httpd.BuildpackYAML
		WebDirectory: c.buildpackYAML.Config.WebDirectory,
		FpmSocket:    "127.0.0.1:9000",
	}

	httpdConfPath := filepath.Join(c.application.Root, "httpd.conf")
	if err := config.ProcessTemplateToFile(config.HttpdConfTemplate, httpdConfPath, httpdCfg); err != nil {
		return err
	}
	return nil
}

func (c Contributor) writePhpFpmConf(layer layers.Layer) error {
	// this path must exist or php-fpm will fail to start
	userIncludePath, err := GetPhpFpmConfPath(c.application.Root)
	if err != nil {
		return err
	}

	phpFpmCfg := config.PhpFpmConfig{
		PhpHome: layer.Root,
		PhpAPI:  os.Getenv("PHP_API"),
		Include: userIncludePath,
		Listen:  "127.0.0.1:9000",
	}

	phpFpmConfPath := filepath.Join(layer.Root, "etc", "php-fpm.conf")
	if err := config.ProcessTemplateToFile(config.PhpFpmConfTemplate, phpFpmConfPath, phpFpmCfg); err != nil {
		return err
	}
	return nil
}

func (c Contributor) initPhp(layer layers.Layer) error {
	if err := c.writePhpIni(layer); err != nil {
		return err
	}

	if err := layer.OverrideSharedEnv("PHPRC", filepath.Join(layer.Root, "etc")); err != nil {
		return err
	}

	if err := layer.OverrideSharedEnv("PHP_INI_SCAN_DIR", filepath.Join(c.application.Root, ".php.ini.d")); err != nil {
		return err
	}

	return nil
}

func (c Contributor) contributeWebApp(layer layers.Layer) error {
	if err := c.initPhp(layer); err != nil {
		return err
	}

	c.logger.SubsequentLine("Using web directory: %s", c.buildpackYAML.Config.WebServer)

	if strings.ToLower(c.buildpackYAML.Config.WebServer) == PhpWebServer {
		c.logger.SubsequentLine("Using PHP built-in server")
		webdir := filepath.Join(c.application.Root, c.buildpackYAML.Config.WebDirectory)
		command := fmt.Sprintf("php -S 0.0.0.0:$PORT -t %s", webdir)

		return c.layers.WriteApplicationMetadata(layers.Metadata{
			Processes: []layers.Process{
				{"web", command},
				{"task", command},
			},
		})
	}

	if strings.ToLower(c.buildpackYAML.Config.WebServer) == ApacheHttpd {
		c.logger.SubsequentLine("Using Apache Web Server")

		if err := c.installProcmgr(layer); err != nil {
			return err
		}

		if err := c.writeHttpdConf(layer); err != nil {
			return err
		}

		if err := c.writePhpFpmConf(layer); err != nil {
			return err
		}

		procsYaml := filepath.Join(layer.Root, "procs.yml")
		procs := procmgr.Procs{
			Processes: map[string]procmgr.Proc{
				"httpd": {
					Command: "httpd",
					Args:    []string{"-f", filepath.Join(c.application.Root, "httpd.conf"), "-k", "start", "-DFOREGROUND"},
				},
				"php-fpm": {
					Command: "php-fpm",
					Args:    []string{"-p", layer.Root, "-y", filepath.Join(layer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(layer.Root, "etc")},
				},
			},
		}

		if err := procmgr.WriteProcs(procsYaml, procs); err != nil {
			return fmt.Errorf("failed to write procs.yml: %s", err)
		}

		return c.layers.WriteApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", fmt.Sprintf("procmgr %s", procsYaml)}}})
	}

	if strings.ToLower(c.buildpackYAML.Config.WebServer) == Nginx {
		// TODO: write out nginx.conf to c.application.Root
		c.logger.SubsequentLine("Using Nginx")
	}

	return nil
}

func (c Contributor) contributeScript(layer layers.Layer) error {
	if err := c.initPhp(layer); err != nil {
		return err
	}

	scriptPath := filepath.Join(c.application.Root, c.buildpackYAML.Config.Script)

	scriptExists, err := helper.FileExists(scriptPath)
	if err != nil {
		return err
	}

	if !scriptExists {
		c.logger.Info("WARNING: `%s` start script not found. App will not start unless you specify a custom start command.", c.buildpackYAML.Config.Script)
	}

	command := fmt.Sprintf("php %s", scriptPath)

	return c.layers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{"web", command},
			{"task", command},
		},
	})
}

// InstallProcmgr adds the procmgr binary to the path
func (c Contributor) installProcmgr(layer layers.Layer) error {
	return helper.CopyFile(c.procmgr, filepath.Join(layer.Root, "bin", "procmgr"))
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	if c.isWebApp {
		c.logger.FirstLine("Configuring PHP Web Application")

		l := c.layers.Layer(WebDependency)
		return l.Contribute(c.metadata, c.contributeWebApp, c.flags()...)
	}


	if c.isScript {
		c.logger.FirstLine("Configuring PHP Script")

		l := c.layers.Layer(ScriptDependency)
		return l.Contribute(c.metadata, c.contributeScript, c.flags()...)
	}

	c.logger.Info("WARNING: Did not detect either a web app or a PHP script to run. App will not start unless you specify a custom start command.")
	return nil
}

func (c Contributor) flags() []layers.Flag {
	return []layers.Flag{
		layers.Launch,
	}
}
