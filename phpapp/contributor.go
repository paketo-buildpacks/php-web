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

package phpapp

import (
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/php-cnb/php"
)

// Contributor represents a PHP contribution by the buildpack
type Contributor struct {
	launchContribution bool
	buildContribution  bool
	phpAppLayer        layers.DependencyLayer
	appRoot            string
	webdir             string
	webserver          string
}

// NewContributor creates a new Contributor instance. willContribute is true if build plan contains "php-binary" dependency, otherwise false.
func NewContributor(context build.Build) (c Contributor, willContribute bool, err error) {
	plan, wantDependency := context.BuildPlan[Dependency]
	if !wantDependency {
		return Contributor{}, false, nil
	}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	buildpackYAML, err := php.LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	version := php.Version(buildpackYAML, context.Buildpack, plan)

	dep, err := deps.Best(Dependency, version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{
		appRoot:     context.Application.Root,
		phpAppLayer: context.Layers.DependencyLayer(dep),
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	return contributor, true, nil
}

// Contribute contributes an expanded PHP to a cache layer.
func (c Contributor) Contribute() error {
	return c.phpAppLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("<TODO> to %s", layer.Root)

		return nil
	}, c.flags()...)
}

func (c Contributor) flags() []layers.Flag {
	var flags []layers.Flag

	if c.buildContribution {
		flags = append(flags, layers.Build, layers.Cache)
	}

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}
