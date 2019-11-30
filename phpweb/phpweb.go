package phpweb

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"gopkg.in/yaml.v2"
)

const (
	// Dependency in the buildplan indicates that this is a web app
	Dependency = "php-web"

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

// Version returns the selected version of PHP using the following precedence:
//
// 1. `php.version` from `buildpack.yml`
// 2. Build Plan Version, if set by composer
// 3. Buildpack Metadata "default_version"
// 4. `*` which should pick latest version
func Version(buildpack buildpack.Buildpack) string {
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
	ServerAdmin  string `yaml:"serveradmin"`
	Redis        Redis  `yaml:"redis"`
}

// Redis represents PHP Redis specific configuration options for `buildpack.yml`
type Redis struct {
	SessionStoreServiceName string `yaml:"session_store_service_name"`
}

// LoadBuildpackYAML reads `buildpack.yml` and PHP specific config options in it
func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")

	buildpackYAML.Config.LibDirectory = "lib"
	buildpackYAML.Config.WebDirectory = "htdocs"
	buildpackYAML.Config.WebServer = ApacheHttpd
	buildpackYAML.Config.ServerAdmin = "admin@localhost"
	buildpackYAML.Config.Redis.SessionStoreServiceName = "redis-sessions"

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

// PickWebDir will select the correct web directory to use
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

// Metadata is used solely for providing `Identity()`
type Metadata struct {
	Name string
	Hash string
}

// Identity provides libcfbuildpack with information to decide if it should contribute
func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}

// Feature is used to add additional features to the CNB
type Feature interface {
	// Name of the feature (for debugging purposes)
	Name() string

	// IsNeeded indicates if this feature is required
	//   frue will enable the feature
	//   false means it's skipped
	IsNeeded() bool

	// EnableFeature will perform the work of enabling the feature
	EnableFeature() error
}
