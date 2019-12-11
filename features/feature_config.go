package features

import (
	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-web-cnb/config"
)

type FeatureConfig struct {
	BpYAML   config.BuildpackYAML
	App      application.Application
	IsWebApp bool
	Logger   logger.Logger
}

// Feature is used to add additional features to the CNB
type Feature interface {
	// Name of the feature (for debugging purposes)
	Name() string

	// IsNeeded indicates if this feature is required
	//   true will enable the feature
	//   false means it's skipped
	IsNeeded() bool

	// EnableFeature will perform the work of enabling the feature
	EnableFeature(layers layers.Layers, currentLayer layers.Layer) error
}
