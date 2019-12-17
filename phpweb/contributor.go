/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package phpweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	layers   layers.Layers
	logger   logger.Logger
	metadata Metadata
	features []features.Feature
}

func generateRandomHash() [32]byte {
	randBuf := make([]byte, 512)
	rand.Read(randBuf)
	return sha256.Sum256(randBuf)
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-script" or "php-web" dependency, otherwise false.
func NewContributor(context build.Build) (Contributor, bool, error) {
	shouldContribute := context.Plans.Has(Dependency)
	if !shouldContribute {
		return Contributor{}, false, nil
	}

	buildpackYAML, err := config.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}
	context.Logger.Debug("Build Pack YAML: %v", buildpackYAML)

	randomHash := generateRandomHash()

	webDir := PickWebDir(buildpackYAML)
	isWebApp, err := SearchForWebApp(context.Application.Root, webDir)
	if err != nil {
		return Contributor{}, false, err
	}

	featureConfig := features.FeatureConfig{
		BpYAML:   buildpackYAML,
		App:      context.Application,
		IsWebApp: isWebApp,
		Logger:   context.Logger,
	}

	contributor := Contributor{
		layers:   context.Layers,
		logger:   context.Logger,
		metadata: Metadata{"PHP Web", hex.EncodeToString(randomHash[:])},
		features: []features.Feature{
			features.NewPhpFeature(featureConfig),
			features.NewPhpWebServerFeature(featureConfig),
			features.NewHttpdFeature(featureConfig),
			features.NewNginxFeature(featureConfig),
			features.NewPhpFpmFeature(featureConfig),
			features.NewRedisFeature(featureConfig, context.Services, buildpackYAML.Config.Redis.SessionStoreServiceName),
			features.NewProcMgrFeature(featureConfig, filepath.Join(context.Buildpack.Root, "bin", "procmgr")),
			features.NewScriptsFeature(featureConfig),
		},
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	return c.layers.Layer(Dependency).Contribute(c.metadata, func(l layers.Layer) error {
		c.logger.Header("Configuring PHP Application")

		// install features
		for _, feature := range c.features {
			if feature.IsNeeded() {
				c.logger.Body("Using feature -- %s", feature.Name())
				err := feature.EnableFeature(c.layers, l)
				if err != nil {
					c.logger.BodyError("Failed %s", err)
					return err
				}
			} else {
				c.logger.Debug("Skipping feature -- %s", feature.Name())
			}
		}

		return nil
	}, c.flags()...)
}

func (c Contributor) flags() []layers.Flag {
	return []layers.Flag{
		layers.Launch,
	}
}
