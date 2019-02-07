package phpweb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"gopkg.in/yaml.v2"
)

const (
	// WebDependency in the buildplan indiates that this is a web app
	WebDependency = "php-web"

	// ScriptDependency in the buildplan indicates that this is a script app
	ScriptDependency = "php-script"

	// Nginx is text user can specify to request Nginx Web Server
	Nginx = "nginx"

	// ApacheHttpd is text user can specify to request Apache Web Server
	ApacheHttpd = "httpd"

	// PhpWebServer is text user can specify to use PHP's built-in Web Server
	PhpWebServer = "php-server"
)

// Version returns the selected version of PHP using the following precedence:
//
// 1. `php.version` from `buildpack.yml`
// 2. Build Plan Version, if set by composer
// 3. Buildpack Metadata "default_version"
// 4. `*` which should pick latest version
func Version(buildpackYAML BuildpackYAML, buildpack buildpack.Buildpack, dependency buildplan.Dependency) string {
	if buildpackYAML.Config.Version != "" {
		return buildpackYAML.Config.Version
	}

	if dependency.Version != "" {
		return dependency.Version
	}

	if version, ok := buildpack.Metadata["default_version"].(string); ok {
		return version
	}

	return "*"
}

// API returns the API string for the given PHP version
func API(version string) string {
	if strings.HasPrefix(version, "7.0") {
		return "20151012"
	} else if strings.HasPrefix(version, "7.1") {
		return "20160303"
	} else if strings.HasPrefix(version, "7.2") {
		return "20170718"
	} else if strings.HasPrefix(version, "7.3") {
		return "20180731"
	} else {
		return ""
	}
}

// BuildpackYAML represents user specified config options through `buildpack.yml`
type BuildpackYAML struct {
	Config Config `yaml:"php"`
}

// Config represents PHP specific configuration options for BuildpackYAML
type Config struct {
	Version      string `yaml:"version"`
	WebServer    string `yaml:"webserver"`
	WebDirectory string `yaml:"webdirectory"`
	Script       string `yaml:"script"`
}

// LoadBuildpackYAML reads `buildpack.yml` and PHP specific config options in it
func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")
	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return BuildpackYAML{}, err
		}

		err = yaml.Unmarshal(contents, &buildpackYAML)
		if err != nil {
			return BuildpackYAML{}, err
		}
	}
	return buildpackYAML, nil
}

// LoadAvailablePHPExtensions locates available extensions and returns the list
func LoadAvailablePHPExtensions(phpLayerRoot string, version string) ([]string, error) {
	extensionFolder := fmt.Sprintf("no-debug-non-zts-%s", API(version))
	extensionPath := filepath.Join(phpLayerRoot, "lib","php", "extensions", extensionFolder, "*.so")
	extensions, err := filepath.Glob(extensionPath)
	if err !=nil {
		return []string{}, err
	}

	for i:=0; i < len(extensions); i++ {
		extensions[i]= strings.Trim(filepath.Base(extensions[i]),".so")
	}

	return extensions, nil
}