package features

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/logger"
	"github.com/buildpack/libbuildpack/platform"
	lbservices "github.com/buildpack/libbuildpack/services"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/services"
)

// MemcachedFeature is used to enable support for session storage via Memcached
type MemcachedFeature struct {
	sessionSupport    MemcachedSessionSupport
	platformRoot      string
	sessionHelperPath string
}

// NewMemcachedFeature an object that Supports Memcached
func NewMemcachedFeature(featureConfig FeatureConfig, srvs services.Services, serviceKey, platformRoot, sessionHelperPath string) MemcachedFeature {
	return MemcachedFeature{
		sessionSupport:    FromExistingMemcachedSessionSupport(featureConfig, srvs, serviceKey),
		platformRoot:      platformRoot,
		sessionHelperPath: sessionHelperPath,
	}
}

// Name of the feature
func (m MemcachedFeature) Name() string {
	return "Memcached Session Support"
}

// IsNeeded determines if this app needs Memcached for storing session data
func (m MemcachedFeature) IsNeeded() bool {
	_, found := m.sessionSupport.FindService()
	return found
}

// EnableFeature will turn on Memcached session storage for PHP
func (m MemcachedFeature) EnableFeature(_ layers.Layers, layer layers.Layer) error {
	err := helper.CopyFile(m.sessionHelperPath, filepath.Join(layer.Root, "bin", "session_helper"))
	if err != nil {
		return err
	}

	err = layer.WriteProfile(
		"0_session_helper.sh",
		SessionHelperScript,
		m.sessionSupport.serviceKey,
		"memcached",
		"memcached",
		m.platformRoot,
		m.sessionSupport.appRoot,
	)
	if err != nil {
		return err
	}

	return nil
}

// MemcachedSessionSupport provides functionality to locate and configure memcached as a session handler
type MemcachedSessionSupport struct {
	appRoot    string
	services   services.Services
	serviceKey string
}

func FromExistingMemcachedSessionSupport(featureConfig FeatureConfig, srvs services.Services, serviceKey string) MemcachedSessionSupport {
	return MemcachedSessionSupport{
		appRoot:    featureConfig.App.Root,
		services:   srvs,
		serviceKey: serviceKey,
	}
}

func NewMemcachedSessionSupport(platformRoot, appRoot string) (MemcachedSessionSupport, error) {
	logger, err := logger.DefaultLogger(platformRoot)
	if err != nil {
		return MemcachedSessionSupport{}, err
	}

	platform, err := platform.DefaultPlatform(platformRoot, logger)
	if err != nil {
		return MemcachedSessionSupport{}, err
	}

	defaultServices, err := lbservices.DefaultServices(platform, logger)
	if err != nil {
		return MemcachedSessionSupport{}, err
	}

	return MemcachedSessionSupport{
		appRoot:    appRoot,
		services:   services.Services{Services: defaultServices},
		serviceKey: "",
	}, nil
}

func (s MemcachedSessionSupport) ConfigureService() error {
	buf := bytes.Buffer{}

	// turn on memcached
	buf.WriteString("extension=memcached.so\n")
	buf.WriteString("extension=igbinary.so\n")
	buf.WriteString("extension=msgpack.so\n")

	// configure PHP to use memcached for sessions
	servers, username, password := s.loadMemcachedProps()
	buf.WriteString(fmt.Sprintf("session.name=PHPSESSIONID\n"))
	buf.WriteString(fmt.Sprintf("session.save_handler=memcached\n"))
	buf.WriteString(fmt.Sprintf("session.save_path=%q\n", servers))
	buf.WriteString("memcached.sess_binary_protocol=On\n")
	buf.WriteString("memcached.sess_persistent=On\n")
	buf.WriteString(fmt.Sprintf("memcached.sess_sasl_username=%q\n", username))
	buf.WriteString(fmt.Sprintf("memcached.sess_sasl_password=%q\n", password))

	// don't use helper.WriteFile because it will mess up the URLencoded values
	filename := filepath.Join(s.appRoot, ".php.ini.d", "memcached-sessions.ini")
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func (s MemcachedSessionSupport) FindService() (services.Credentials, bool) {
	// This is here for backwards compatibility. Previously the buildpack would
	// look for a service with a specific name or a name given by the user
	for _, service := range s.services.Services {
		if service.BindingName == s.serviceKey {
			return service.Credentials, true
		}
	}

	// If not found, we just look for anything providing Memcached, to be more flexible, or return nil
	return s.services.FindServiceCredentials("memcached")
}

func (s MemcachedSessionSupport) loadMemcachedProps() (servers string, username string, password string) {
	creds, found := s.FindService()
	if found {
		var found bool
		if servers, found = creds["servers"].(string); !found {
			servers = "127.0.0.1"
		}

		if username, found = creds["username"].(string); !found {
			username = ""
		}

		if password, found = creds["password"].(string); !found {
			password = ""
		}

		return servers, username, password
	}

	// shouldn't happen, as this method should only run if there is a service bound
	//   and when that happens there should be creds
	return "127.0.0.1", "", ""
}
