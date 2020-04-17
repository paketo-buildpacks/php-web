package phpweb

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/php-web/config"

	"github.com/cloudfoundry/libcfbuildpack/buildpack"
)

const (
	// Dependency in the buildplan indicates that this is a web app
	Dependency = "php-web"
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

// PickWebDir will select the correct web directory to use
func PickWebDir(buildpackYAML config.BuildpackYAML) string {
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
