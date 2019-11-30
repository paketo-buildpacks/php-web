package features

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/services"
)

// RedisFeature is used to enable support for session storage via Redis
type RedisFeature struct {
	appRoot    string
	services   services.Services
	serviceKey string
}

// NewRedisFeature an object that Supports Redis
func NewRedisFeature(app application.Application, srvs services.Services, serviceKey string) RedisFeature {
	return RedisFeature{
		appRoot:    app.Root,
		services:   srvs,
		serviceKey: serviceKey,
	}
}

// Name of the feature
func (r RedisFeature) Name() string {
	return "Redis Session Support"
}

// IsNeeded determines if this app needs Redis for storing session data
func (r RedisFeature) IsNeeded() bool {
	creds := r.findService()
	return creds != nil
}

// EnableFeature will turn on Redis session storage for PHP
func (r RedisFeature) EnableFeature() error {
	buf := bytes.Buffer{}

	// turn on redis & igbinary extensions (redis needs igbinary)
	buf.WriteString("extension=redis.so\n")
	buf.WriteString("extension=igbinary.so\n")

	// configure PHP to use redis for sessions
	savePath := r.formatRedisURL()
	buf.WriteString(fmt.Sprintf("session.name=PHPSESSIONID\n"))
	buf.WriteString(fmt.Sprintf("session.save_handler=redis\n"))
	buf.WriteString(fmt.Sprintf("session.save_path=%s\n", savePath))

	// don't use helper.WriteFile because it will mess up the URLencoded values
	filename := filepath.Join(r.appRoot, ".php.ini.d", "redis-sessions.ini")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func (r RedisFeature) findService() services.Credentials {
	// This is here for backwards compatibility. Previously the buildpack would
	// look for a service with a specific name or a name given by the user
	for _, service := range r.services.Services {
		if service.BindingName == r.serviceKey {
			return service.Credentials
		}
	}

	// If not found, we just look for anything providing Redis, to be more flexible, or return nil
	creds, _ := r.services.FindServiceCredentials("redis")
	return creds
}

func (r RedisFeature) formatRedisURL() string {
	host, port, password := r.loadRedisProps()
	redisURL := fmt.Sprintf("tcp://%s:%d", host, port)
	if password != "" {
		redisURL = fmt.Sprintf("%s?auth=%s", redisURL, url.QueryEscape(password))
	}
	return redisURL
}

func (r RedisFeature) loadRedisProps() (host string, port int, password string) {
	creds := r.findService()
	if creds != nil {
		var found bool
		if host, found = creds["host"].(string); !found {
			if host, found = creds["hostname"].(string); !found {
				host = "127.0.0.1"
			}
		}

		if port, found = creds["port"].(int); !found {
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
