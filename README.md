# caddydev
Tool for developing custom [Caddy](http://caddyserver.com) middleware

### Installation
```
$ go get github.com/caddyserver/caddydev
```

### Usage
caddydev creates and starts a custom Caddy on the fly with the currently developed middleware integrated.
```shell
$ caddydev help
Usage:
	caddydev [-c|-h|help] [caddy flags]

	-c=middleware.json - Path to config file.
	-h=false - show this usage.
	help - alias for -h=true
	caddy flags - flags to pass to caddy.
```

### Config
caddydev requires a config file named `middleware.json`

Sample config
```json
{
  "name": "Hello",
  "description": "Hello middleware says hello",
  "import": "github.com/abiosoft/hello-caddy",
  "repository": "https://github.com/abiosoft/hello-caddy",
  "directive": "hello",
  "after": "git"
}
```
Config | Details
-------|--------
name | Name of the middleware
description | What does your middleware do
import | go import path
repository | middleware repository
directive | keyword to register middleware in Caddyfile
after (optional) | priority of middleware. What directive should it be placed after.
