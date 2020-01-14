package features_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/libcfbuildpack/services"
	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitMemcached(t *testing.T) {
	spec.Run(t, "Memcached", testMemcached, spec.Report(report.Terminal{}))
}

func testMemcached(t *testing.T, when spec.G, it spec.S) {

	var (
		factory *test.BuildFactory
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	memcachedFeatureFactory := func(svcs services.Services) features.MemcachedFeature {
		return features.NewMemcachedFeature(
			features.FeatureConfig{
				BpYAML:   config.BuildpackYAML{},
				App:      factory.Build.Application,
				IsWebApp: true,
			},
			svcs,
			"memcached-sessions",
			factory.Build.Platform.Root,
			filepath.Join(factory.Build.Buildpack.Root, "bin", "session_helper"),
		)
	}

	when("IsNeeded", func() {
		when("memcached is present", func() {
			it.Before(func() {
				factory = test.NewBuildFactory(t)
			})

			it("is detected when name is `memcached`", func() {
				factory.AddService("memcached", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				})
				r := memcachedFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})

			it("is detected when name is not `memcached` but there is a `memcached` tag", func() {
				factory.AddService("something", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				}, "memcached")
				r := memcachedFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})

			it("is detected when name is `memcached-sessions`", func() {
				factory.AddService("memcached-sessions", services.Credentials{
					"username": "fake1",
					"password": "fake2",
				})
				r := memcachedFeatureFactory(factory.Build.Services)

				Expect(r.IsNeeded()).To(BeTrue())
			})
		})

		when("memcached isn't present", func() {
			it.Before(func() {
				factory = test.NewBuildFactory(t)
			})

			it("is not detected", func() {
				r := memcachedFeatureFactory(factory.Build.Services)
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
			r := memcachedFeatureFactory(factory.Build.Services)
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
  --binding-name "memcached-sessions" \
  --search-term "memcached" \
  --session-driver "memcached" \
  --platform-root %q \
  --app-root %q
`,
					factory.Build.Platform.Root,
					factory.Build.Application.Root,
				),
			))
		})
	})

	when("MemcachedSessionSupport", func() {
		var sessionSupport features.MemcachedSessionSupport

		when("FindService", func() {
			when("there is no given service key", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("memcached", services.Credentials{
						"username": "fake1",
						"password": "fake2",
					})

					sessionSupport = features.FromExistingMemcachedSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"",
					)
				})

				it("finds the credentials for the memcached service", func() {
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
					factory.AddService("memcached-session", services.Credentials{
						"username": "fake1",
						"password": "fake2",
					})

					sessionSupport = features.FromExistingMemcachedSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"memcached-session",
					)
				})

				it("finds the credentials for the memcached service", func() {
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

					sessionSupport = features.FromExistingMemcachedSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"memcached-session",
					)
				})

				it("finds the credentials for the memcached service", func() {
					_, found := sessionSupport.FindService()
					Expect(found).To(BeFalse())
				})
			})
		})

		when("ConfigureService", func() {
			when("the service does not contain credentials", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("memcached-session", services.Credentials{})

					sessionSupport = features.FromExistingMemcachedSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"memcached-session",
					)
				})

				it("is enabled with defaults", func() {
					Expect(sessionSupport.ConfigureService()).To(Succeed())

					iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "memcached-sessions.ini")
					Expect(iniPath).To(BeARegularFile())

					contents, err := ioutil.ReadFile(iniPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(ContainSubstring("extension=memcached.so"))
					Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
					Expect(string(contents)).To(ContainSubstring("session.save_handler=memcached"))
					Expect(string(contents)).To(ContainSubstring(`session.save_path="127.0.0.1"`))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_binary_protocol=On"))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_persistent=On"))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_sasl_username=\"\""))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_sasl_password=\"\""))
				})
			})

			when("the service does contain credentials", func() {
				it.Before(func() {
					factory = test.NewBuildFactory(t)
					factory.AddService("memcached-sessions", services.Credentials{
						"servers":  "192.168.0.1:1234",
						"username": "user-1",
						"password": "fake!@#$%\"^&*()-={]}[?><,./;':",
					})

					sessionSupport = features.FromExistingMemcachedSessionSupport(
						features.FeatureConfig{
							BpYAML:   config.BuildpackYAML{},
							App:      factory.Build.Application,
							IsWebApp: true,
						},
						factory.Build.Services,
						"memcached-session",
					)
				})

				it("is enabled with service values", func() {
					Expect(sessionSupport.ConfigureService()).To(Succeed())

					iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "memcached-sessions.ini")
					Expect(iniPath).To(BeARegularFile())

					contents, err := ioutil.ReadFile(iniPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(ContainSubstring("extension=memcached.so"))
					Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
					Expect(string(contents)).To(ContainSubstring("session.save_handler=memcached"))
					Expect(string(contents)).To(ContainSubstring(`session.save_path="192.168.0.1:1234"`))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_binary_protocol=On"))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_persistent=On"))
					Expect(string(contents)).To(ContainSubstring("memcached.sess_sasl_username=\"user-1\""))
					Expect(string(contents)).To(ContainSubstring(`memcached.sess_sasl_password="fake!@#$%\"^&*()-={]}[?><,./;':"`))
				})
			})
		})
	})
}
