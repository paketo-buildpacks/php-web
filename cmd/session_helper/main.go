package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cloudfoundry/php-web-cnb/features"
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

	search, err := features.NewRedisSessionSupport(platformRoot, appRoot)
	if err != nil {
		log.Fatalln("NewRedisSessionSupport:", err)
	}
	//TODO: enable memcached support
	err = search.ConfigureService()
	if err != nil {
		log.Fatalln("Search:", err)
	}
}
