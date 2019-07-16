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

// NginxConfTemplate is the template string for a nginx.conf file
const NginxConfTemplate = `daemon     off;
error_log  stderr  notice;

worker_processes  auto;
events {
    worker_connections  1024;
}

http {
	types {
		text/html                             html htm shtml;
		text/css                              css;
		text/xml                              xml;
		image/gif                             gif;
		image/jpeg                            jpeg jpg;
		application/javascript                js;
		application/atom+xml                  atom;
		application/rss+xml                   rss;
	
		text/mathml                           mml;
		text/plain                            txt;
		text/vnd.sun.j2me.app-descriptor      jad;
		text/vnd.wap.wml                      wml;
		text/x-component                      htc;
	
		image/png                             png;
		image/tiff                            tif tiff;
		image/vnd.wap.wbmp                    wbmp;
		image/x-icon                          ico;
		image/x-jng                           jng;
		image/x-ms-bmp                        bmp;
		image/svg+xml                         svg svgz;
		image/webp                            webp;
	
		application/font-woff                 woff;
		application/java-archive              jar war ear;
		application/json                      json;
		application/mac-binhex40              hqx;
		application/msword                    doc;
		application/pdf                       pdf;
		application/postscript                ps eps ai;
		application/rtf                       rtf;
		application/vnd.ms-excel              xls;
		application/vnd.ms-fontobject         eot;
		application/vnd.ms-powerpoint         ppt;
		application/vnd.wap.wmlc              wmlc;
		application/vnd.google-earth.kml+xml  kml;
		application/vnd.google-earth.kmz      kmz;
		application/x-7z-compressed           7z;
		application/x-cocoa                   cco;
		application/x-java-archive-diff       jardiff;
		application/x-java-jnlp-file          jnlp;
		application/x-makeself                run;
		application/x-perl                    pl pm;
		application/x-pilot                   prc pdb;
		application/x-rar-compressed          rar;
		application/x-redhat-package-manager  rpm;
		application/x-sea                     sea;
		application/x-shockwave-flash         swf;
		application/x-stuffit                 sit;
		application/x-tcl                     tcl tk;
		application/x-x509-ca-cert            der pem crt;
		application/x-xpinstall               xpi;
		application/xhtml+xml                 xhtml;
		application/zip                       zip;
	
		application/octet-stream              bin exe dll;
		application/octet-stream              deb;
		application/octet-stream              dmg;
		application/octet-stream              iso img;
		application/octet-stream              msi msp msm;
	
		application/vnd.openxmlformats-officedocument.wordprocessingml.document    docx;
		application/vnd.openxmlformats-officedocument.spreadsheetml.sheet          xlsx;
		application/vnd.openxmlformats-officedocument.presentationml.presentation  pptx;
	
		audio/midi                            mid midi kar;
		audio/mpeg                            mp3;
		audio/ogg                             ogg;
		audio/x-m4a                           m4a;
		audio/x-realaudio                     ra;
	
		video/3gpp                            3gpp 3gp;
		video/mp4                             mp4;
		video/mpeg                            mpeg mpg;
		video/quicktime                       mov;
		video/webm                            webm;
		video/x-flv                           flv;
		video/x-m4v                           m4v;
		video/x-mng                           mng;
		video/x-ms-asf                        asx asf;
		video/x-ms-wmv                        wmv;
		video/x-msvideo                       avi;
	}

    default_type       application/octet-stream;
    sendfile           on;
    keepalive_timeout  65;
    gzip               on;
    port_in_redirect   off;
    root               {{.AppRoot}}/{{.WebDirectory}};
    index              index.php index.html;
    server_tokens      off;

    log_format common '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent';
    log_format extended '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent vcap_request_id=$http_x_vcap_request_id';
    access_log  /dev/stdout  extended;

    # set $https only when SSL is actually used.
    map $http_x_forwarded_proto $proxy_https {
        https on;
    }

    # setup the scheme to use on redirects
    map $http_x_forwarded_proto $redirect_scheme {
        default http;
        http http;
        https https;
    }

    upstream php_fpm {
        server unix:{{.FpmSocket}};
    }

	server {
        listen       {{"{{"}}env "PORT"{{"}}"}};
        server_name  _;

        fastcgi_temp_path      /tmp/nginx_fastcgi 1 2;
        client_body_temp_path  /tmp/nginx_client_body 1 2;
        proxy_temp_path        /tmp/nginx_proxy 1 2;

        real_ip_header         x-forwarded-for;
        set_real_ip_from       10.0.0.0/8;
        real_ip_recursive      on;

        # Some basic cache-control for static files to be sent to the browser
        location ~* \.(?:ico|css|js|gif|jpeg|jpg|png)$ {
            expires         max;
            add_header      Pragma public;
            add_header      Cache-Control "public, must-revalidate, proxy-revalidate";
        }

        # Deny hidden files (.htaccess, .htpasswd, .DS_Store).
        location ~ /\. {
            deny            all;
            access_log      off;
            log_not_found   off;
        }

        location ~ .*\.php$ {
            try_files $uri =404;

			fastcgi_param  QUERY_STRING       $query_string;
			fastcgi_param  REQUEST_METHOD     $request_method;
			fastcgi_param  CONTENT_TYPE       $content_type;
			fastcgi_param  CONTENT_LENGTH     $content_length;

			fastcgi_param  SCRIPT_NAME        $fastcgi_script_name;
			fastcgi_param  REQUEST_URI        $request_uri;
			fastcgi_param  DOCUMENT_URI       $document_uri;
			fastcgi_param  DOCUMENT_ROOT      $document_root;
			fastcgi_param  SERVER_PROTOCOL    $server_protocol;
			fastcgi_param  HTTPS              $proxy_https if_not_empty;

			fastcgi_param  GATEWAY_INTERFACE  CGI/1.1;
			fastcgi_param  SERVER_SOFTWARE    nginx/$nginx_version;

			fastcgi_param  REMOTE_ADDR        $remote_addr;
			fastcgi_param  REMOTE_PORT        $remote_port;
			fastcgi_param  SERVER_ADDR        $server_addr;
			fastcgi_param  SERVER_PORT        $server_port;
			fastcgi_param  SERVER_NAME        $server_name;
			fastcgi_param HTTP_PROXY "";

            fastcgi_param   SCRIPT_FILENAME $document_root$fastcgi_script_name;
            fastcgi_pass    php_fpm;
        }

        # support folder redirects with and without trailing slashes
        location ~ "^(.*)[^/]$" {
            if (-d $document_root$uri) {
                rewrite ^ $redirect_scheme://$http_host$uri/ permanent;
            }
        }
	}
}
`
