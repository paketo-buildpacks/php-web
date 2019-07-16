package phpweb

import (
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

// BuildpackYAML represents user specified config options through `buildpack.yml`
type BuildpackYAML struct {
	Config Config `yaml:"php"`
}

// Config represents PHP specific configuration options for BuildpackYAML
type Config struct {
	Version      string `yaml:"version"`
	WebServer    string `yaml:"webserver"`
	WebDirectory string `yaml:"webdirectory"`
	LibDirectory string `yaml:"libdirectory"`
	Script       string `yaml:"script"`
}

// LoadBuildpackYAML reads `buildpack.yml` and PHP specific config options in it
func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")

	buildpackYAML.Config.LibDirectory = "lib"
	buildpackYAML.Config.WebDirectory = "htdocs"
	buildpackYAML.Config.WebServer = ApacheHttpd
	buildpackYAML.Config.Script = "app.php"

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
func LoadAvailablePHPExtensions() ([]string, error) {
	extensions, err := filepath.Glob(filepath.Join(os.Getenv("PHP_EXTENSION_DIR"), "*"))
	if err != nil {
		return []string{}, err
	}

	for i := 0; i < len(extensions); i++ {
		extensions[i] = strings.Trim(filepath.Base(extensions[i]), ".so")
	}

	return extensions, nil
}

// GetPhpFpmConfPath will look to see if a user has specified custom PHP-FPM config & if so return the path. Returns an empty string if not specified.
func GetPhpFpmConfPath(appRoot string) (string, error) {
	userIncludePath := filepath.Join(appRoot, ".php.fpm.d", "*.conf")
	matches, err := filepath.Glob(userIncludePath)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		userIncludePath = ""
	}

	return userIncludePath, nil
}

type Metadata struct {
	Name string
	Hash string
}

// Identity provides libcfbuildpack with information to decide if it should contribute
func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}
