package features

import (
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/paketo-buildpacks/php-web/config"
	"github.com/paketo-buildpacks/php-web/procmgr"
	"os"
	"path/filepath"
	"strings"
)

type NginxFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
	isWebApp bool
}

func NewNginxFeature(featureConfig FeatureConfig) NginxFeature {
	return NginxFeature{
		bpYAML: featureConfig.BpYAML,
		app:    featureConfig.App,
		isWebApp: featureConfig.IsWebApp,
	}
}

func (p NginxFeature) IsNeeded() bool {
	return strings.ToLower(p.bpYAML.Config.WebServer) == config.Nginx && p.isWebApp
}

func (p NginxFeature) Name() string {
	return "Nginx"
}

func (p NginxFeature) EnableFeature(commonLayers layers.Layers, currentLayer layers.Layer) error {
	if err := p.writeConfig(currentLayer); err != nil {
		return err
	}

	return p.updateProcs(currentLayer)
}

func (p NginxFeature) writeConfig(currentLayer layers.Layer) error {
	cfg := config.NginxConfig{
		AppRoot:      p.app.Root,
		WebDirectory: p.bpYAML.Config.WebDirectory,
		FpmSocket:    filepath.Join(currentLayer.Root, "php-fpm.socket"),
	}
	template := config.NginxConfTemplate
	confPath := filepath.Join(p.app.Root, "nginx.conf")
	return config.ProcessTemplateToFile(template, confPath, cfg)
}

func (p NginxFeature) updateProcs(layer layers.Layer) error {
	err := os.MkdirAll(layer.Root, 0755)
	if err != nil {
		return err
	}

	procsYaml := filepath.Join(layer.Root, "procs.yml")
	procs := procmgr.Procs{
		Processes: map[string]procmgr.Proc{
			"nginx": procmgr.Proc{
				Command: "nginx",
				Args:    []string{"-p", p.app.Root, "-c", filepath.Join(p.app.Root, "nginx.conf")},
			},
		},
	}

	return procmgr.AppendOrUpdateProcs(procsYaml, procs)
}
