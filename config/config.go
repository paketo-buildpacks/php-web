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

package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/cloudfoundry/libcfbuildpack/helper"
)

const (
	// Nginx is text user can specify to request Nginx Web Server
	Nginx = "nginx"

	// ApacheHttpd is text user can specify to request Apache Web Server
	ApacheHttpd = "httpd"

	// PhpWebServer is text user can specify to use PHP's built-in Web Server
	PhpWebServer = "php-server"
)

var (
	// DefaultCliScripts is the script used when one is not provided in buildpack.yml
	DefaultCliScripts = []string{"app.php", "main.php", "run.php", "start.php"}
)

// ProcessTemplateToFile writes out a specific template to the given file name
func ProcessTemplateToFile(templateBody string, outputPath string, data interface{}) error {
	template, err := template.New(filepath.Base(outputPath)).Parse(templateBody)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	err = template.Execute(&b, data)
	if err != nil {
		return err
	}

	return helper.WriteFileFromReader(outputPath, 0644, &b)
}

// HttpdConfig supplies values for templated httpd.conf
type HttpdConfig struct {
	ServerAdmin          string
	DisableHTTPSRedirect bool
	AppRoot              string
	WebDirectory         string
	FpmSocket            string
}

// NginxConfig supplies values for templated nginx.conf
type NginxConfig struct {
	DisableHTTPSRedirect bool
	AppRoot              string
	WebDirectory         string
	FpmSocket            string
}

// PhpIniConfig supplies values for templated php.ini
type PhpIniConfig struct {
	AppRoot        string
	LibDirectory   string
	PhpHome        string
	PhpAPI         string
	Extensions     []string
	ZendExtensions []string
}

// PhpFpmConfig supplies values for templated php-fpm.conf
type PhpFpmConfig struct {
	PhpHome string
	PhpAPI  string
	Include string
	Listen  string
}

// BuildpackYAML represents user specified config options through `buildpack.yml`
type BuildpackYAML struct {
	Config Config `yaml:"php"`
}

// Config represents PHP specific configuration options for BuildpackYAML
type Config struct {
	Version             string    `yaml:"version"`
	WebServer           string    `yaml:"webserver"`
	WebDirectory        string    `yaml:"webdirectory"`
	LibDirectory        string    `yaml:"libdirectory"`
	Script              string    `yaml:"script"`
	ServerAdmin         string    `yaml:"serveradmin"`
	EnableHTTPSRedirect bool      `yaml:"enable_https_redirect"`
	Redis               Redis     `yaml:"redis"`
	Memcached           Memcached `yaml:"memcached"`
}

// Redis represents PHP Redis specific configuration options for `buildpack.yml`
type Redis struct {
	SessionStoreServiceName string `yaml:"session_store_service_name"`
}

// Memcached represents PHP Memcached specific configuration options for `buildpack.yml`
type Memcached struct {
	SessionStoreServiceName string `yaml:"session_store_service_name"`
}

// LoadBuildpackYAML reads `buildpack.yml` and PHP specific config options in it
func LoadBuildpackYAML(appRoot string) (BuildpackYAML, map[string]string, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")

	buildpackYAML.Config.EnableHTTPSRedirect = true
	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, map[string]string{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, map[string]string{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return BuildpackYAML{}, map[string]string{}, err
		}

		err = yaml.Unmarshal(contents, &buildpackYAML)
		if err != nil {
			return BuildpackYAML{}, map[string]string{}, err
		}
	}
	configMapping, finalBuildpackYAML := yamlConfig(buildpackYAML)
	return finalBuildpackYAML, configMapping, nil
}

// Get map of config fields set by buildpack.yml and also set default values for any empty fields
func yamlConfig(buildpackYAML BuildpackYAML) (map[string]string, BuildpackYAML) {
	fieldMapping := map[string]string{}

	if buildpackYAML.Config.Version != "" {
		fieldMapping["version"] = "BP_PHP_VERSION"
	}
	if buildpackYAML.Config.LibDirectory != "" {
		fieldMapping["lib directory"] = "BP_PHP_LIB_DIR"
	} else {
		buildpackYAML.Config.LibDirectory = "lib"
	}
	if buildpackYAML.Config.WebDirectory != "" {
		fieldMapping["web directory"] = "BP_PHP_WEB_DIR"
	} else {
		buildpackYAML.Config.WebDirectory = "htdocs"
	}
	if buildpackYAML.Config.WebServer != "" {
		fieldMapping["web server"] = "BP_PHP_SERVER"
	} else {
		buildpackYAML.Config.WebServer = PhpWebServer
	}
	if buildpackYAML.Config.ServerAdmin != "" {
		fieldMapping["server admin"] = "BP_PHP_SERVER_ADMIN"
	} else {
		buildpackYAML.Config.ServerAdmin = "admin@localhost"
	}
	if buildpackYAML.Config.Redis.SessionStoreServiceName != "" {
		fieldMapping["redis session store service name"] = "service binding"
	} else {
		buildpackYAML.Config.Redis.SessionStoreServiceName = "redis-sessions"
	}
	if buildpackYAML.Config.Memcached.SessionStoreServiceName != "" {
		fieldMapping["memcache session store service name"] = "service binding"
	} else {
		buildpackYAML.Config.Memcached.SessionStoreServiceName = "memcached-sessions"
	}
	if buildpackYAML.Config.Script != "" {
		fieldMapping["scripts"] = "Procfile"
	}
	return fieldMapping, buildpackYAML
}

func PickWebDir(buildpackYAML BuildpackYAML) string {
	if buildpackYAML.Config.WebDirectory != "" {
		return buildpackYAML.Config.WebDirectory
	}

	return "htdocs"
}

// SearchForWebApp looks to see if this application is a PHP web app
func SearchForWebApp(appRoot string, webdir string) (bool, error) {
	matchList, err := filepath.Glob(filepath.Join(appRoot, webdir, "*.php"))
	if err != nil {
		return false, err
	}

	if len(matchList) > 0 {
		return true, nil
	}
	return false, nil
}
