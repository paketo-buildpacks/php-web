package features_test

import (
	"github.com/cloudfoundry/libcfbuildpack/services"
	"github.com/cloudfoundry/php-web-cnb/config"
	"github.com/cloudfoundry/php-web-cnb/features"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/layers"

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
		)
	}

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

		it("is enabled with defaults", func() {
			layer := factory.Build.Layers.Layer("layer-1")
			r := redisFeatureFactory(factory.Build.Services)

			err := r.EnableFeature(layers.Layers{}, layer)
			Expect(err).ToNot(HaveOccurred())

			iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "redis-sessions.ini")
			Expect(iniPath).To(BeARegularFile())

			contents, err := ioutil.ReadFile(iniPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("extension=redis.so"))
			Expect(string(contents)).To(ContainSubstring("extension=igbinary.so"))
			Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
			Expect(string(contents)).To(ContainSubstring("session.save_handler=redis"))
			Expect(string(contents)).To(ContainSubstring("session.save_path=tcp://127.0.0.1:6379"))
		})

		it("is enabled with service values", func() {
			factory.AddService("redis-sessions", services.Credentials{
				"host":     "192.168.0.1",
				"port":     65309,
				"password": "fake!@#$%^&*()-={]}[?><,./;':",
			})
			r := redisFeatureFactory(factory.Build.Services)

			layer := factory.Build.Layers.Layer("layer-1")
			err := r.EnableFeature(layers.Layers{}, layer)
			Expect(err).ToNot(HaveOccurred())

			iniPath := filepath.Join(factory.Build.Application.Root, ".php.ini.d", "redis-sessions.ini")
			Expect(iniPath).To(BeARegularFile())

			contents, err := ioutil.ReadFile(iniPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("extension=redis.so"))
			Expect(string(contents)).To(ContainSubstring("extension=igbinary.so"))
			Expect(string(contents)).To(ContainSubstring("session.name=PHPSESSIONID"))
			Expect(string(contents)).To(ContainSubstring("session.save_handler=redis"))
			Expect(string(contents)).To(ContainSubstring("session.save_path=tcp://192.168.0.1:65309?auth=fake%21%40%23%24%25%5E%26%2A%28%29-%3D%7B%5D%7D%5B%3F%3E%3C%2C.%2F%3B%27%3A"))
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
}
