# caddydev

[![Join the chat at https://gitter.im/caddyserver/caddydev](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/caddyserver/caddydev?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
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
##### 2. Start caddydev.
```shell
$ caddydev --source github.com/abiosoft/hello-caddy hello
Starting caddy...
0.0.0.0:2015
```
##### 3. Test it.
```
$ curl localhost:2015
Hello, I'm a caddy middleware
```
[github.com/abiosoft/hello-caddy](https://github.com/abiosoft/hello-caddy) can be the template for your new middleware. Follow the link to learn more.

### Usage
caddydev creates and starts a custom Caddy on the fly with the currently developed middleware integrated.
```
$ caddydev -h
Usage: caddydev [options] directive [caddy flags] [go [build flags]]

options:
  -s, --source="."   Source code directory or go get path.
  -a, --after=""     Priority. After which directive should our new directive be placed.
  -u, --update=false Pull latest caddy source code before building.
  -o, --output=""    Path to save custom build. If set, the binary will only be generated, not executed.
                     Set GOOS, GOARCH, GOARM environment variables to generate for other platforms.
  -h, --help=false   Show this usage.

directive:
  directive of the middleware being developed.

caddy flags:
  flags to pass to the resulting custom caddy binary.

go build flags:
  flags to pass to 'go build' while building custom binary prefixed with 'go'.
  go keyword is used to differentiate caddy flags from go build flags.
  e.g. go -race -x -v.
```

### Note
caddydev is in active development and can still change significantly.

### Disclaimer
This software is provided as-is and you assume all risk.