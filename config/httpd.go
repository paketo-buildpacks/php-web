/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

// HttpdConfTemplate is the template string for a httpd.conf file
const HttpdConfTemplate = `ServerRoot "${SERVER_ROOT}"
Listen ${PORT}
ServerAdmin "{{.ServerAdmin}}"
ServerName "0.0.0.0"
DocumentRoot "{{.AppRoot}}/{{.WebDirectory}}"

# Load only modules required for PHP
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule env_module modules/mod_env.so
LoadModule setenvif_module modules/mod_setenvif.so
LoadModule dir_module modules/mod_dir.so
LoadModule mime_module modules/mod_mime.so
LoadModule reqtimeout_module modules/mod_reqtimeout.so
LoadModule unixd_module modules/mod_unixd.so
LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule proxy_module modules/mod_proxy.so
LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so
LoadModule remoteip_module modules/mod_remoteip.so
LoadModule rewrite_module modules/mod_rewrite.so
LoadModule filter_module modules/mod_filter.so
LoadModule deflate_module modules/mod_deflate.so
LoadModule headers_module modules/mod_headers.so

# Secure Directory Permissions
<Directory />
    AllowOverride none
    Require all denied
</Directory>

<Directory "{{.AppRoot}}/{{.WebDirectory}}">
    Options SymLinksIfOwnerMatch
    AllowOverride All
    Require all granted
</Directory>

<Files ".ht*">
    Require all denied
</Files>

# set up mime types
<IfModule mime_module>
    TypesConfig conf/mime.types
    AddType application/x-compress .Z
    AddType application/x-gzip .gz .tgz
</IfModule>

# Deflate Support
<IfModule filter_module>
    <IfModule deflate_module>
        AddOutputFilterByType DEFLATE text/html text/plain text/xml text/css text/javascript application/javascript
    </IfModule>
</IfModule>

# Log everything to STDOUT/STDERR & log CF specific info
ErrorLog "/proc/self/fd/2"
LogLevel info
<IfModule log_config_module>
    LogFormat "%a %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined
    LogFormat "%a %l %u %t \"%r\" %>s %b" common
    LogFormat "%a %l %u %t \"%r\" %>s %b vcap_request_id=%{X-Vcap-Request-Id}i peer_addr=%{c}a" extended
    <IfModule logio_module>
      LogFormat "%a %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\" %I %O" combinedio
    </IfModule>
    CustomLog "/proc/self/fd/1" extended
</IfModule>

# configure event MPM
<IfModule mpm_event_module>
    StartServers             3
    MinSpareThreads         75
    MaxSpareThreads        250
    ThreadsPerChild         25
    MaxRequestWorkers      400
    MaxConnectionsPerChild   0
</IfModule>

# Defaults
Timeout 60
KeepAlive On
MaxKeepAliveRequests 100
KeepAliveTimeout 5
UseCanonicalName Off
UseCanonicalPhysicalPort Off
AccessFileName .htaccess
ServerTokens Prod
ServerSignature Off
HostnameLookups Off
EnableMMAP Off
EnableSendfile On
RequestReadTimeout header=20-40,MinRate=500 body=20,MinRate=500

#
# Adjust IP Address based on header set by proxy
#
RemoteIpHeader x-forwarded-for
RemoteIpInternalProxy 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16

#
# Set HTTPS environment variable if we came in over secure
#  channel.
SetEnvIf x-forwarded-proto https HTTPS=on

# Talk to PHP via FCGI & php-fpm
DirectoryIndex index.php index.html index.htm

Define fcgi-listener fcgi://{{.FpmSocket}}{{.AppRoot}}/{{.WebDirectory}}

<Proxy "${fcgi-listener}">
    # Noop ProxySet directive, disablereuse=On is the default value.
    # If we don't have a ProxySet, this <Proxy> isn't handled
    # correctly and everything breaks.

    # NOTE: Setting retry to avoid cached HTTP 503 (See https://www.pivotaltracker.com/story/show/103840940)
    ProxySet disablereuse=On retry=0
</Proxy>

<Directory "{{.AppRoot}}/{{.WebDirectory}}">
  <Files *.php>
      <If "-f %{REQUEST_FILENAME}"> # make sure the file exists so that if not, Apache will show its 404 page and not FPM
          SetHandler proxy:fcgi://{{.FpmSocket}}
      </If>
  </Files>
</Directory>

RequestHeader unset Proxy early
`
