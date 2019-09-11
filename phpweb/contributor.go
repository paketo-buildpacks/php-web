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
	buildpackYAML BuildpackYAML
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

	isWebApp := context.Plans.Has(WebDependency)
	isScript := context.Plans.Has(ScriptDependency)
	randomHash := generateRandomHash()

	contributor := Contributor{
		application:   context.Application,
		layers:        context.Layers,
		logger:        context.Logger,
		buildpackYAML: buildpackYAML,
		isWebApp:      isWebApp,
		isScript:      isScript,
		procmgr:       filepath.Join(context.Buildpack.Root, "bin", "procmgr"),
		metadata:      Metadata{"PHP Web", hex.EncodeToString(randomHash[:])},
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	if c.isWebApp {
		c.logger.Header("Configuring PHP Web Application")

		l := c.layers.Layer(WebDependency)
		return l.Contribute(c.metadata, c.contributeWebApp, c.flags()...)
	}

	if c.isScript {
		c.logger.Header("Configuring PHP Script")

		l := c.layers.Layer(ScriptDependency)
		return l.Contribute(c.metadata, c.contributeScript, c.flags()...)
	}

	c.logger.BodyWarning("WARNING: Did not detect either a web app or a PHP script to run. App will not start unless you specify a custom start command.")
	return nil
}

func (c Contributor) contributeWebApp(layer layers.Layer) error {
	if err := c.initPhp(layer); err != nil {
		return err
	}

	c.logger.Body("Requested web server: %s", c.buildpackYAML.Config.WebServer)

	webServerName := strings.ToLower(c.buildpackYAML.Config.WebServer)
	if webServerName == PhpWebServer {
		c.logger.Body("Using PHP built-in server")
		webdir := filepath.Join(c.application.Root, c.buildpackYAML.Config.WebDirectory)
		command := fmt.Sprintf("php -S 0.0.0.0:$PORT -t %s", webdir)

		return c.layers.WriteApplicationMetadata(layers.Metadata{
			Processes: []layers.Process{
				{"web", command, false},
				{"task", command, false},
			},
		})
	} else if webServerName == ApacheHttpd {
		c.logger.Body("Using Apache Web Server")

		process := procmgr.Proc{
			Command: "httpd",
			Args:    []string{"-f", filepath.Join(c.application.Root, "httpd.conf"), "-k", "start", "-DFOREGROUND"},
		}

		return c.contributeWebServer(layer, webServerName, process)
	} else if webServerName == Nginx {
		c.logger.Body("Using Nginx Web Server")

		process := procmgr.Proc{
			Command: "nginx",
			Args:    []string{"-p", c.application.Root, "-c", filepath.Join(c.application.Root, "nginx.conf")},
		}

		return c.contributeWebServer(layer, webServerName, process)
	}

	c.logger.BodyWarning("WARNING: Did not install requested web server: %s. Requested server is unavailable. Start command is not being provided.", webServerName)
	return nil
}

func (c Contributor) contributeWebServer(layer layers.Layer, name string, webProc procmgr.Proc) error {
	if err := c.installProcmgr(layer); err != nil {
		return err
	}

	if err := c.writeServerConf(layer, name); err != nil {
		return err
	}

	if err := c.writePhpFpmConf(layer, name); err != nil {
		return err
	}

	procsYaml := filepath.Join(layer.Root, "procs.yml")
	procs := procmgr.Procs{
		Processes: map[string]procmgr.Proc{
			name: webProc,
			"php-fpm": {
				Command: "php-fpm",
				Args:    []string{"-p", layer.Root, "-y", filepath.Join(layer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(layer.Root, "etc")},
			},
		},
	}

	if err := procmgr.WriteProcs(procsYaml, procs); err != nil {
		return fmt.Errorf("failed to write procs.yml: %s", err)
	}

	return c.layers.WriteApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", fmt.Sprintf("procmgr %s", procsYaml), false}}})

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
			{"web", command, false},
			{"task", command, false},
		},
	})
}

func (c Contributor) writeServerConf(layer layers.Layer, name string) error {
	var (
		cfg      interface{}
		template string
	)

	if name == ApacheHttpd {
		cfg = config.HttpdConfig{
			ServerAdmin:  c.buildpackYAML.Config.ServerAdmin,
			AppRoot:      c.application.Root,
			WebDirectory: c.buildpackYAML.Config.WebDirectory,
			FpmSocket:    "127.0.0.1:9000",
		}
		template = config.HttpdConfTemplate
	} else if name == Nginx {
		cfg = config.NginxConfig{
			AppRoot:      c.application.Root,
			WebDirectory: c.buildpackYAML.Config.WebDirectory,
			FpmSocket:    filepath.Join(layer.Root, "php-fpm.socket"),
		}
		template = config.NginxConfTemplate
	}

	confPath := filepath.Join(c.application.Root, fmt.Sprintf("%s.conf", name))
	return config.ProcessTemplateToFile(template, confPath, cfg)
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

func (c Contributor) writePhpFpmConf(layer layers.Layer, server string) error {
	// this path must exist or php-fpm will fail to start
	userIncludePath, err := GetPhpFpmConfPath(c.application.Root)
	if err != nil {
		return err
	}

	phpFpmCfg := config.PhpFpmConfig{
		PhpHome: layer.Root,
		PhpAPI:  os.Getenv("PHP_API"),
		Include: userIncludePath,
	}

	if server == ApacheHttpd {
		phpFpmCfg.Listen = "127.0.0.1:9000"
	} else {
		phpFpmCfg.Listen = filepath.Join(layer.Root, "php-fpm.socket")
	}

	phpFpmConfPath := filepath.Join(layer.Root, "etc", "php-fpm.conf")
	return config.ProcessTemplateToFile(config.PhpFpmConfTemplate, phpFpmConfPath, phpFpmCfg)
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

// InstallProcmgr adds the procmgr binary to the path
func (c Contributor) installProcmgr(layer layers.Layer) error {
	return helper.CopyFile(c.procmgr, filepath.Join(layer.Root, "bin", "procmgr"))
}

func (c Contributor) flags() []layers.Flag {
	return []layers.Flag{
		layers.Launch,
	}
}
