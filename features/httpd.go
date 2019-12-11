package features

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/php-web-cnb/procmgr"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/php-web-cnb/config"
)


type HttpdFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
	isWebApp bool
}

func NewHttpdFeature(featureConfig FeatureConfig) HttpdFeature {
	return HttpdFeature{
		bpYAML: featureConfig.BpYAML,
		app:    featureConfig.App,
		isWebApp: featureConfig.IsWebApp,
	}
}

func (p HttpdFeature) IsNeeded() bool {
	return strings.ToLower(p.bpYAML.Config.WebServer) == config.ApacheHttpd && p.isWebApp
}

func (p HttpdFeature) Name() string {
	return "Apache Web Server"
}

func (p HttpdFeature) EnableFeature(commonLayers layers.Layers, currentLayer layers.Layer) error {
	if err := p.writeConfig(); err != nil {
		return err
	}

	return p.updateProcs(currentLayer)
}

func (p HttpdFeature) writeConfig() error {

	cfg := config.HttpdConfig{
		ServerAdmin:  p.bpYAML.Config.ServerAdmin,
		AppRoot:      p.app.Root,
		WebDirectory: p.bpYAML.Config.WebDirectory,
		FpmSocket:    "127.0.0.1:9000",
	}
	template := config.HttpdConfTemplate
	confPath := filepath.Join(p.app.Root, "httpd.conf")
	return config.ProcessTemplateToFile(template, confPath, cfg)
}

func (p HttpdFeature) updateProcs(layer layers.Layer) error {
	err := os.MkdirAll(layer.Root, 0755)
	if err != nil {
		return err
	}

	procsYaml := filepath.Join(layer.Root, "procs.yml")
	procs := procmgr.Procs{
		Processes: map[string]procmgr.Proc{
			"httpd": procmgr.Proc{
				Command: "httpd",
				Args:    []string{"-f", filepath.Join(p.app.Root, "httpd.conf"), "-k", "start", "-DFOREGROUND"},
			},
		},
	}

	return procmgr.AppendOrUpdateProcs(procsYaml, procs)
}
