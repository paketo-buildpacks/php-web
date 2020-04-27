# PHP Web Cloud Native Buildpack

The Paketo PHP Web Buildpack is a Cloud Native Buildpack V3 that configures PHP applications to run.

This buildpack is designed to work in collaboration with other buildpacks which do things like provide PHP binaries.
(e.g. [PHP Dist CNB](https://github.com/paketo-buildpacks/php-dist))

## Detection

The detection phase passes if either of the following conditions hold true:

- `<APPLICATION_ROOT>/<WEBDIR>/*.php` exists
- `<APPLICATION_ROOT>/**/*.php` exists

## Build

Looks at `buildpack.yml` for `php.webserver`, if
  - `php-server`, contribute a web process type using `php -S`
  - `httpd`, generate a suitable `httpd.conf`
  - `nginx`, generate a suitable `nginx.conf`

## Integration

The PHP Web CNB is the last in the standard chain of PHP CNBs. It provides `php-web`
as a dependency, but currently there's no scenario we can imagine that you would use
a downstream buildpack to require this dependency. If a user likes to include some other
functionality (like a monitoring tool or a db driver), it can be done independent of
the PHP Web CNB without requiring a dependency of it.

## To Package

To package this buildpack for consumption:

```bash
$ ./scripts/package.sh
```

This builds the buildpack's Go source using GOOS=linux by default. You can supply another value as the first argument to package.sh.

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

 ## `buildpack.yml` Configurations

 ```yaml
 php:
  # this allows you to specify a version constaint for the `php` dependency
  # any valid semver constaints (e.g. 10.* and 10.1.*) are also acceptable
  version: 10.1.x

  # text user can specify to use PHP's built-in Web Server
  # default: php-server
  webserver: php-server

  # directory where web app code is stored
  # default: htdocs
  webdirectory: htdocs

  # directory where library code is stored
  # default: lib
  libdirectory: lib

  # no default
  script:

  # default: admin@localhost
  serveradmin: admin@localhost

  # default: redis-sessions
  redis:
    session_store_service_name: redis-sessions

  # default: memcached-sessions
  memcached:
    session_store_service_name: memcached-sessions
```
