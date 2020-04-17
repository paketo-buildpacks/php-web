package main

import (
	"flag"
	"log"
	"strings"

	"github.com/paketo-buildpacks/php-web/features"
)

func main() {
	var bindingName, searchTerm, sessionDriver, platformRoot, appRoot string

	flag.StringVar(&bindingName, "binding-name", "", "binding name used in search")
	flag.StringVar(&searchTerm, "search-term", "", "fuzzy search term, used if binding name not found")
	flag.StringVar(&sessionDriver, "session-driver", "", "session handler to configure: redis or memcached")
	flag.StringVar(&platformRoot, "platform-root", "", "platform root for the CNB")
	flag.StringVar(&appRoot, "app-root", "", "application root")
	flag.Parse()

	if bindingName == "" || searchTerm == "" || platformRoot == "" || appRoot == "" {
		log.Fatalln("binding-name, search-term, platform-root and app-root are required")
	}

	sessionDriver = strings.ToLower(sessionDriver)
	if sessionDriver != "redis" && sessionDriver != "memcached" {
		log.Fatalln("session-driver [", sessionDriver, "] not valid. Valid options are: redis or memcached")
	}

	var search features.SessionConfigurer
	var err error

	if sessionDriver == "redis" {
		search, err = features.NewRedisSessionSupport(platformRoot, appRoot)
	} else if sessionDriver == "memcached" {
		search, err = features.NewMemcachedSessionSupport(platformRoot, appRoot)
	}
	if err != nil {
		log.Fatalln("NewSessionConfigurer:", err)
	}

	err = search.ConfigureService()
	if err != nil {
		log.Fatalln("Search:", err)
	}
}
