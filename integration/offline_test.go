package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testOffline(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		pack       occam.Pack
		docker     occam.Docker
	)

	SetDefaultEventuallyTimeout(10 * time.Second)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when offline", func() {
		var (
			image     occam.Image
			container occam.Container
			name      string
			source    string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("creates a working OCI image that serves web pages using builtin php server", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "php_app"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithNoPull().
				WithBuildpacks(phpDistOfflineURI, phpWebOfflineURI).
				WithNetwork("none").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs.String()).To(ContainSubstring(buildpackInfo.Buildpack.Name))
			Expect(logs.String()).NotTo(ContainSubstring("Downloading"))

			container, err = docker.Container.Run.WithEnv(map[string]string{"PORT":"8080"}).Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable(), logs.String())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.php", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())
			Expect(string(content)).To(ContainSubstring("phpinfo()"))
		})

		it("creates a working OCI image that serves web pages using nginx server", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app_nginx"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithNoPull().
				WithBuildpacks(nginxOfflineURI, phpDistOfflineURI, phpWebOfflineURI).
				WithNetwork("none").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs.String()).To(ContainSubstring(buildpackInfo.Buildpack.Name))
			Expect(logs.String()).NotTo(ContainSubstring("Downloading"))

			container, err = docker.Container.Run.WithEnv(map[string]string{"PORT":"8080"}).Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable(), logs.String())

			// later TODO: wait until the web server connects to php
			time.Sleep(5 * time.Second)

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.php?date", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())
			Expect(string(content)).To(ContainSubstring("SUCCESS"))
		})

		it("creates a working OCI image that serves web pages using httpd server", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app_httpd"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithNoPull().
				WithBuildpacks(httpdOfflineURI, phpDistOfflineURI, phpWebOfflineURI).
				WithNetwork("none").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs.String()).To(ContainSubstring(buildpackInfo.Buildpack.Name))
			Expect(logs.String()).NotTo(ContainSubstring("Downloading"))

			container, err = docker.Container.Run.WithEnv(map[string]string{"PORT":"8080"}).Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable(), logs.String())

			// later TODO: wait until the web server connects to php
			time.Sleep(5 * time.Second)

			response, err := http.Get(fmt.Sprintf("http://localhost:%s/index.php?date", container.HostPort()))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Body.Close()).To(Succeed())
			Expect(string(content)).To(ContainSubstring("SUCCESS"))
		})
	})
}
