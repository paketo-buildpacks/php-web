# PHP App Cloud Native Buildpack

The Paketo PHP App Buildpack is a Cloud Native Buildpack V3 that configures PHP applications to run.

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

The PHP App CNB provides `php-web` as a dependency. Downstream buildpacks can require the php-web dependency, however
this buildpack signifies the end of the PHP group build processes, so any extension to this could be included in other
independent buildpacks. Requiring `php-web` is not a workflow that is supported.

## To Package

To package this buildpack for consumption:

```bash
$ ./scripts/package.sh
```

This builds the buildpack's Go source using GOOS=linux by default. You can supply another value as the first argument to package.sh.

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
