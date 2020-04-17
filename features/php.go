package features

import (
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/paketo-buildpacks/php-web/config"
	"os"
	"path/filepath"
)

type PhpFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
}

func NewPhpFeature(featureConfig FeatureConfig) PhpFeature {
	return PhpFeature{
		bpYAML: featureConfig.BpYAML,
		app:    featureConfig.App,
	}
}

func (p PhpFeature) IsNeeded() bool {
	return true
}

func (p PhpFeature) Name() string {
	return "PHP"
}

func (p PhpFeature) EnableFeature(commonLayers layers.Layers, currentLayer layers.Layer) error {
	if err := p.writePhpIni(currentLayer); err != nil {
		return err
	}

	if err := currentLayer.OverrideSharedEnv("PHPRC", filepath.Join(currentLayer.Root, "etc")); err != nil {
		return err
	}

	return currentLayer.OverrideSharedEnv("PHP_INI_SCAN_DIR", filepath.Join(p.app.Root, ".php.ini.d"))
}

func (p PhpFeature) writePhpIni(layer layers.Layer) error {
	phpIniCfg := config.PhpIniConfig{
		AppRoot:      p.app.Root,
		LibDirectory: p.bpYAML.Config.LibDirectory,
		PhpHome:      os.Getenv("PHP_HOME"),
		PhpAPI:       os.Getenv("PHP_API"),
	}
	phpIniPath := filepath.Join(layer.Root, "etc", "php.ini")
	return config.ProcessTemplateToFile(config.PhpIniTemplate, phpIniPath, phpIniCfg)
}
