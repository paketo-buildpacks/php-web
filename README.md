# PHP App Cloud Native Buildpack

The Cloud Foundry PHP App Buildpack is a Cloud Native Buildpack V3 that configures PHP applications to run.

This buildpack is designed to work in collaboration with other buildpacks which do things like provide PHP binaries.

## Detection

The detection phase passes if

- `<APPLICATION_ROOT>/<WEBDIR>/*.php` exists
  - Contributes `php-binary` to the build plan
  - Contributes `php-binary.metadata.launch = true` to the build plan
  - Contributes `php-web` to the build plan

if no web app is found then detection can still pass if

- `<APPLICATION_ROOT>/**/*.php` exists
  - Contributes `php-binary` to the build plan
  - Contributes `php-binary.metadata.launch = true` to the build plan
  - Contributes `php-script` to the build plan

## Build

If the build plan contains

- `php-script`
  - contributes a `web` process type to run the script
  - contributes a `task` process type to run the script

- `php-web`
  - looks at `buildpack.yml` for `php.webserver`, if
    - `php-server`, contribute a web process type using `php -S`
    - `httpd`, generate a suitable `httpd.conf`
    - `nginx`, generate a suitable `nginx.conf`

## To Package

To package this buildpack for consumption:

```bash
$ ./scripts/package.sh
```

This builds the buildpack's Go source using GOOS=linux by default. You can supply another value as the first argument to package.sh.

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0