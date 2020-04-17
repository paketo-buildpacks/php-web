package features

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/paketo-buildpacks/php-web/config"
)

type PhpWebServerFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
	isWebApp bool
}

func NewPhpWebServerFeature(featureConfig FeatureConfig) PhpWebServerFeature {
	return PhpWebServerFeature{
		bpYAML: featureConfig.BpYAML,
		app:    featureConfig.App,
		isWebApp: featureConfig.IsWebApp,
	}
}

func (p PhpWebServerFeature) IsNeeded() bool {
	return strings.ToLower(p.bpYAML.Config.WebServer) == config.PhpWebServer && p.isWebApp
}

func (p PhpWebServerFeature) Name() string {
	return "PHP Web Server"
}

func (p PhpWebServerFeature) EnableFeature(commonLayers layers.Layers, _ layers.Layer) error {
	webdir := filepath.Join(p.app.Root, p.bpYAML.Config.WebDirectory)
	command := fmt.Sprintf("php -S 0.0.0.0:$PORT -t %s", webdir)

	return commonLayers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{Type: "web", Command: command, Direct: false},
			{Type: "task", Command: command, Direct: false},
		},
	})
}
