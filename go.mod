module github.com/cloudfoundry/php-app-cnb

require (
	github.com/buildpack/libbuildpack v1.9.0
	github.com/cloudfoundry/dagger v0.0.0-20190108154828-8e5ab63c9f02
	github.com/cloudfoundry/libcfbuildpack v1.37.0
	github.com/cloudfoundry/php-cnb v0.0.1
	github.com/onsi/gomega v1.4.3
	github.com/sclevine/spec v1.2.0
)

replace github.com/cloudfoundry/php-cnb v0.0.1 => ../php-cnb/
