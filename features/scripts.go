package features

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/paketo-buildpacks/php-web/config"
)

type ScriptsFeature struct {
	bpYAML   config.BuildpackYAML
	app      application.Application
	isWebApp bool
	logger   logger.Logger
}

func NewScriptsFeature(featureConfig FeatureConfig) ScriptsFeature {
	return ScriptsFeature{
		bpYAML:   featureConfig.BpYAML,
		app:      featureConfig.App,
		isWebApp: featureConfig.IsWebApp,
		logger:   featureConfig.Logger,
	}
}

func (p ScriptsFeature) IsNeeded() bool {
	return !p.isWebApp
}

func (p ScriptsFeature) Name() string {
	return "Scripts"
}

func (p ScriptsFeature) EnableFeature(commonLayers layers.Layers, currentLayer layers.Layer) error {
	if p.bpYAML.Config.Script == "" {
		for _, possible := range config.DefaultCliScripts {
			exists, err := helper.FileExists(filepath.Join(p.app.Root, possible))
			if err != nil {
				return err
				// skip and continue
			}

			if exists {
				p.bpYAML.Config.Script = possible
				break
			}
		}

		if p.bpYAML.Config.Script == "" {
			p.bpYAML.Config.Script = "app.php"
			p.logger.BodyWarning("Buildpack could not find a file to execute. Either set php.script in buildpack.yml or include one of these files [%s]", strings.Join(config.DefaultCliScripts, ", "))
		}
	}

	scriptPath := filepath.Join(p.app.Root, p.bpYAML.Config.Script)
	command := fmt.Sprintf("php %s", scriptPath)

	return commonLayers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{Type: "web", Command: command, Direct: false},
			{Type: "task", Command: command, Direct: false},
		},
	})
}
