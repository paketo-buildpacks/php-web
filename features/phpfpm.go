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

type PhpFpmFeature struct {
	bpYAML config.BuildpackYAML
	app    application.Application
	isWebApp bool
}

func NewPhpFpmFeature(featureConfig FeatureConfig) PhpFpmFeature {
	return PhpFpmFeature{
		bpYAML: featureConfig.BpYAML,
		app:    featureConfig.App,
		isWebApp: featureConfig.IsWebApp,
	}
}

func (p PhpFpmFeature) IsNeeded() bool {
	serverName := strings.ToLower(p.bpYAML.Config.WebServer)
	serverMap := map[string]bool {
		config.Nginx: true,
		config.ApacheHttpd: true,
	}
	_, exists := serverMap[serverName]
 	return exists && p.isWebApp
}

func (p PhpFpmFeature) Name() string {
	return "PhpFpm"
}

func (p PhpFpmFeature) EnableFeature(commonLayers layers.Layers, currentLayer layers.Layer) error {
	if err := p.writeConfig(currentLayer); err != nil {
		return err
	}

	return p.updateProcs(currentLayer)
}

func (p PhpFpmFeature) writeConfig(currentLayer layers.Layer) error {
	// this path must exist or php-fpm will fail to start
	userIncludePath, err := p.getPhpFpmConfPath()
	if err != nil {
		return err
	}

	cfg := config.PhpFpmConfig{
		PhpHome: currentLayer.Root,
		PhpAPI:  os.Getenv("PHP_API"),
		Include: userIncludePath,
	}

	if p.bpYAML.Config.WebServer == config.ApacheHttpd {
		cfg.Listen = "127.0.0.1:9000"
	} else {
		cfg.Listen = filepath.Join(currentLayer.Root, "php-fpm.socket")
	}

	template := config.PhpFpmConfTemplate
	confPath := filepath.Join(currentLayer.Root, "etc", "php-fpm.conf")
	return config.ProcessTemplateToFile(template, confPath, cfg)
}

func (p PhpFpmFeature) updateProcs(layer layers.Layer) error {
	err := os.MkdirAll(layer.Root, 0755)
	if err != nil {
		return err
	}

	procsYaml := filepath.Join(layer.Root, "procs.yml")
	procs := procmgr.Procs{
		Processes: map[string]procmgr.Proc{
			"php-fpm": {
				Command: "php-fpm",
				Args:    []string{"-p", layer.Root, "-y", filepath.Join(layer.Root, "etc", "php-fpm.conf"), "-c", filepath.Join(layer.Root, "etc")},
			},
		},
	}

	return procmgr.AppendOrUpdateProcs(procsYaml, procs)
}

// getPhpFpmConfPath will look to see if a user has specified custom PHP-FPM config & if so return the path. Returns an empty string if not specified.
func (p PhpFpmFeature) getPhpFpmConfPath() (string, error) {
	userIncludePath := filepath.Join(p.app.Root, ".php.fpm.d", "*.conf")
	matches, err := filepath.Glob(userIncludePath)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		userIncludePath = ""
	}

	return userIncludePath, nil
}
