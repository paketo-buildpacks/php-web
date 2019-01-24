package phpweb

const (
	// WebDependency in the buildplan indiates that this is a web app
	WebDependency = "php-web"

	// ScriptDependency in the buildplan indicates that this is a script app
	ScriptDependency = "php-script"

	// Nginx is text user can specify to request Nginx Web Server
	Nginx = "nginx"

	// ApacheHttpd is text user can specify to request Apache Web Server
	ApacheHttpd = "httpd"

	// PhpWebServer is text user can specify to use PHP's built-in Web Server
	PhpWebServer = "php-server"
)
