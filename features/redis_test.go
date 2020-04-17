package features_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/libcfbuildpack/services"
	"github.com/paketo-buildpacks/php-web/config"
	"github.com/paketo-buildpacks/php-web/features"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitRedis(t *testing.T) {
	spec.Run(t, "Redis", testRedis, spec.Report(report.Terminal{}))
}

func testRedis(t *testing.T, when spec.G, it spec.S) {

	var (
		factory *test.BuildFactory
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	redisFeatureFactory := func(svcs services.Services) features.RedisFeature {
		return features.NewRedisFeature(
			features.FeatureConfig{
				BpYAML:   config.BuildpackYAML{},
				App:      factory.Build.Application,
				IsWebApp: true,
			},
			svcs,
			"redis-sessions",
			factory.Build.Platform.Root,
			filepath.Join(factory.Build.Buildpack.Root, "bin", "session_helper"),
		)
	}

	when("IsNeeded", func() {
		when("redis is present", func() {
			it.Before(func() {
				factory = test.NewBuildFactory(t)
			})

			it("is detected when name is `redis`", func() {
				factory.AddService("redis", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				})
				r := redisFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})

			it("is detected when name is not `redis` but there is a `redis` tag", func() {
				factory.AddService("something", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				}, "redis")
				r := redisFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})

			it("is detected when name is `redis-sessions`", func() {
				factory.AddService("redis-sessions", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				})
				r := redisFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})
		})

		when("redis isn't present", func() {
			it.Before(func() {
				factory = test.NewBuildFactory(t)
			})

			it("is not detected", func() {
				r := redisFeatureFactory(factory.Build.Services)
				Expect(r.IsNeeded()).To(BeFalse())
			})
		})
	})

	when("EnableFeature", func() {
		var layer layers.Layer

		it.Before(func() {
			factory = test.NewBuildFactory(t)
			layer = factory.Build.Layers.Layer("test")

			err := os.MkdirAll(filepath.Join(factory.Build.Buildpack.Root, "bin"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(
				filepath.Join(factory.Build.Buildpack.Root, "bin", "session_helper"),
				[]byte("session-helper-contents"),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		it("writes a profile.d script to run the session_helper", func() {
			r := redisFeatureFactory(factory.Build.Services)
			Expect(r.EnableFeature(factory.Build.Layers, layer)).To(Succeed())

			sessionHelperFile, err := os.Open(filepath.Join(layer.Root, "bin", "session_helper"))
			Expect(err).NotTo(HaveOccurred())

			contents, err := ioutil.ReadAll(sessionHelperFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("session-helper-contents"))

			info, err := sessionHelperFile.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()).To(Equal(os.FileMode(0644)))

			script, err := ioutil.ReadFile(filepath.Join(layer.Root, "profile.d", "0_session_helper.sh"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(script)).To(Equal(
				fmt.Sprintf(`#!/bin/bash
session_helper \
  --binding-name "redis-sessions" \
  --search-term "redis" \
  --session-driver "redis" \
  --platform-root %q \
  --app-root %q
`,
					factory.Build.Platform.Root,
					factory.Build.Application.Root,
				),
			))
		})
	})

	when("RedisSessionSupport", func() {
		var sessionSupport features.RedisSessionSupport

		when("FindService", func() {
			when("there is no given service key", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("redis", services.Credentials{
						"username": "fake1",
						"password": "fake2",
					})

					sessionSupport = features.FromExistingRedisSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"",
					)
				})

				it("finds the credentials for the redis service", func() {
					service, found := sessionSupport.FindService()
					Expect(found).To(BeTrue())
					Expect(service).To(Equal(services.Credentials{
						"username": "fake1",
						"password": "fake2",
					}))
				})
			})

			when("there is a service key given", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("redis-session", services.Credentials{
						"username": "fake1",
						"password": "fake2",
					})

					sessionSupport = features.FromExistingRedisSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"redis-session",
					)
				})

				it("finds the credentials for the redis service", func() {
					service, found := sessionSupport.FindService()
					Expect(found).To(BeTrue())
					Expect(service).To(Equal(services.Credentials{
						"username": "fake1",
						"password": "fake2",
					}))
				})
			})

			when("there is no matching service", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)

					sessionSupport = features.FromExistingRedisSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"redis-session",
					)
				})

				it("finds the credentials for the redis service", func() {
					_, found := sessionSupport.FindService()
					Expect(found).To(BeFalse())
				})
			})
		})

		when("ConfigureService", func() {
			when("the service does not contain credentials", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("redis-session", services.Credentials{})

					sessionSupport = features.FromExistingRedisSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"redis-session",
					)
				})

				it("is enabled with defaults", func() {
					Expect(sessionSupport.ConfigureService()).To(Succeed())

					iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "redis-sessions.ini")
					Expect(iniPath).To(BeARegularFile())

					contents, err := ioutil.ReadFile(iniPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(ContainSubstring("extension=redis.so"))
					Expect(string(contents)).To(ContainSubstring("extension=igbinary.so"))
					Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
					Expect(string(contents)).To(ContainSubstring("session.save_handler=redis"))
					Expect(string(contents)).To(ContainSubstring("session.save_path=\"tcp://127.0.0.1:6379\""))
				})
			})

			when("the service does contain credentials", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("redis-sessions", services.Credentials{
						"host":     "192.168.0.1",
						"port":     float64(65309), // simulate how JSON handles numbers as float64
						"password": "fake!@#$%\"^&*()-={]}[?><,./;':",
					})

					sessionSupport = features.FromExistingRedisSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"redis-session",
					)
				})

				it("is enabled with service values", func() {
					Expect(sessionSupport.ConfigureService()).To(Succeed())

					iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "redis-sessions.ini")
					Expect(iniPath).To(BeARegularFile())

					contents, err := ioutil.ReadFile(iniPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(ContainSubstring("extension=redis.so"))
					Expect(string(contents)).To(ContainSubstring("extension=igbinary.so"))
					Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
					Expect(string(contents)).To(ContainSubstring("session.save_handler=redis"))
					Expect(string(contents)).To(ContainSubstring("session.save_path=\"tcp://192.168.0.1:65309?auth=fake%21%40%23%24%25%22%5E%26%2A%28%29-%3D%7B%5D%7D%5B%3F%3E%3C%2C.%2F%3B%27%3A\""))
				})
			})
		})
	})
}
