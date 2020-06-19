package features

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/buildpack/libbuildpack/logger"
	"github.com/buildpack/libbuildpack/platform"
	"github.com/cloudfoundry/libcfbuildpack/layers"

	lbservices "github.com/buildpack/libbuildpack/services"
	"github.com/cloudfoundry/libcfbuildpack/services"
)

const SessionHelperScript = `#!/bin/bash
session_helper \
  --binding-name %q \
  --search-term %q \
  --session-driver %q \
  --platform-root %q \
  --app-root %q
`

// RedisFeature is used to enable support for session storage via Redis
type RedisFeature struct {
	sessionSupport    RedisSessionSupport
	platformRoot      string
	sessionHelperPath string
}

// NewRedisFeature an object that Supports Redis
func NewRedisFeature(featureConfig FeatureConfig, srvs services.Services, serviceKey, platformRoot, sessionHelperPath string) RedisFeature {
	return RedisFeature{
		sessionSupport:    FromExistingRedisSessionSupport(featureConfig, srvs, serviceKey),
		platformRoot:      platformRoot,
		sessionHelperPath: sessionHelperPath,
	}
}

// Name of the feature
func (r RedisFeature) Name() string {
	return "Redis Session Support"
}

// IsNeeded determines if this app needs Redis for storing session data
func (r RedisFeature) IsNeeded() bool {
	_, found := r.sessionSupport.FindService()
	return found
}

// EnableFeature will turn on Redis session storage for PHP
func (r RedisFeature) EnableFeature(_ layers.Layers, layer layers.Layer) error {
	err := helper.CopyFile(r.sessionHelperPath, filepath.Join(layer.Root, "bin", "session_helper"))
	if err != nil {
		return err
	}

	err = layer.WriteProfile(
		"0_session_helper.sh",
		SessionHelperScript,
		r.sessionSupport.serviceKey,
		"redis",
		"redis",
		r.platformRoot,
		r.sessionSupport.appRoot,
	)
	if err != nil {
		return err
	}

	return nil
}

// RedisSessionSupport provides functionality to locate and configure redis as a session handler
type RedisSessionSupport struct {
	appRoot    string
	services   services.Services
	serviceKey string
}

func FromExistingRedisSessionSupport(featureConfig FeatureConfig, srvs services.Services, serviceKey string) RedisSessionSupport {
	return RedisSessionSupport{
		appRoot:    featureConfig.App.Root,
		services:   srvs,
		serviceKey: serviceKey,
	}
}

func NewRedisSessionSupport(platformRoot, appRoot string) (RedisSessionSupport, error) {
	logger, err := logger.DefaultLogger(platformRoot)
	if err != nil {
		return RedisSessionSupport{}, err
	}

	platform, err := platform.DefaultPlatform(platformRoot, logger)
	if err != nil {
		return RedisSessionSupport{}, err
	}

	defaultServices, err := lbservices.DefaultServices(platform, logger)
	if err != nil {
		return RedisSessionSupport{}, err
	}

	return RedisSessionSupport{
		appRoot:    appRoot,
		services:   services.Services{Services: defaultServices},
		serviceKey: "",
	}, nil
}

func (s RedisSessionSupport) ConfigureService() error {
	buf := bytes.Buffer{}

	// turn on redis & igbinary extensions (redis needs igbinary)
	buf.WriteString("extension=redis.so\n")
	buf.WriteString("extension=igbinary.so\n")

	// configure PHP to use redis for sessions
	savePath := s.formatRedisURL()
	buf.WriteString("session.name=PHPSESSIONID\n")
	buf.WriteString("session.save_handler=redis\n")
	buf.WriteString(fmt.Sprintf("session.save_path=%q\n", savePath))

	// don't use helper.WriteFile because it will mess up the URLencoded values
	filename := filepath.Join(s.appRoot, ".php.ini.d", "redis-sessions.ini")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func (s RedisSessionSupport) FindService() (services.Credentials, bool) {
	// This is here for backwards compatibility. Previously the buildpack would
	// look for a service with a specific name or a name given by the user
	for _, service := range s.services.Services {
		if service.BindingName == s.serviceKey {
			return service.Credentials, true
		}
	}

	// If not found, we just look for anything providing Redis, to be more flexible, or return nil
	return s.services.FindServiceCredentials("redis")
}

func (s RedisSessionSupport) formatRedisURL() string {
	host, port, password := s.loadRedisProps()
	redisURL := fmt.Sprintf("tcp://%s:%d", host, port)
	if password != "" {
		redisURL = fmt.Sprintf("%s?auth=%s", redisURL, url.QueryEscape(password))
	}
	return redisURL
}

func (s RedisSessionSupport) loadRedisProps() (host string, port int, password string) {
	creds, found := s.FindService()
	if found {
		var found bool
		if host, found = creds["host"].(string); !found {
			if host, found = creds["hostname"].(string); !found {
				host = "127.0.0.1"
			}
		}

		credPort, found := creds["port"].(float64)
		if found {
			port = int(credPort)
		} else {
			port = 6379
		}

		if password, found = creds["password"].(string); !found {
			password = ""
		}

		return host, port, password
	}

	// shouldn't happen, as this method should only run if there is a service bound
	//   and when that happens there should be creds
	return "127.0.0.1", 6379, ""
}
