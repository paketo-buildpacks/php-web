package features

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/paketo-buildpacks/php-web/config"
)

type ProcMgrFeature struct {
	bpYAML      config.BuildpackYAML
	procMgrPath string
	isWebApp bool
}

func NewProcMgrFeature(featureConfig FeatureConfig, procMgrPath string) ProcMgrFeature {
	return ProcMgrFeature{
		bpYAML:      featureConfig.BpYAML,
		isWebApp:   featureConfig.IsWebApp,
		procMgrPath: procMgrPath,
	}
}

func (p ProcMgrFeature) IsNeeded() bool {
	return p.bpYAML.Config.WebServer == config.Nginx || p.bpYAML.Config.WebServer == config.ApacheHttpd
}

func (p ProcMgrFeature) Name() string {
	return "ProcMgr"
}

func (p ProcMgrFeature) EnableFeature(currentLayers layers.Layers, currentLayer layers.Layer) error {
	err := helper.CopyFile(p.procMgrPath, filepath.Join(currentLayer.Root, "bin", "procmgr"))
	if err != nil {
		return err
	}

	procsYaml := filepath.Join(currentLayer.Root, "procs.yml")

	return currentLayers.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{Type: "web", Command: fmt.Sprintf("procmgr %s", procsYaml), Direct: false},
		},
	})

}
