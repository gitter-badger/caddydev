# caddydev
Tool for developing custom [Caddy](http://caddyserver.com) middleware.

### Installation
```shell
$ go get github.com/caddyserver/caddydev
```

### Middleware Development
##### 1. Pull hello middleware.
```shell
$ go get github.com/abiosoft/hello-caddy
```
##### 2. Navigate to the source directory.
```shell
$ cd $GOPATH/src/github.com/abiosoft/hello-caddy
```
##### 3. Start caddydev.
```shell
$ caddydev
Starting caddy...
0.0.0.0:2015
```
##### 4. Test it.
```
$ curl localhost:2015
Hello, I'm a caddy middleware
```
[github.com/abiosoft/hello-caddy](https://github.com/abiosoft/hello-caddy) can be the template for your new middleware. Follow the link to learn more.

### Usage
caddydev creates and starts a custom Caddy on the fly with the currently developed middleware integrated.
```
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
  "after": "gzip"
}
```
Config | Details
-------|--------
name | Name of the middleware
description | What does your middleware do
import | go get compatible import path
repository | source code repository
directive | keyword to register middleware in Caddyfile
after (optional) | priority of middleware (for development purpose only). What directive should it be placed after.

### Note
caddydev is in active development and can still change significantly.

### License
This program is copyrighted, proprietary property and, as such, no license is granted for commercial use or redistribution.

### Disclaimer
This software is provided as-is and you assume all risk.