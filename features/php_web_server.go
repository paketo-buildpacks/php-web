package features

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/php-web-cnb/config"
)

type PhpWebServerFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
}

func NewPhpWebServerFeature(app application.Application, bpYAML config.BuildpackYAML) PhpWebServerFeature {
	return PhpWebServerFeature{
		bpYAML: bpYAML,
		app:    app,
	}
}

func (p PhpWebServerFeature) IsNeeded() bool {
	return strings.ToLower(p.bpYAML.Config.WebServer) == config.PhpWebServer
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
