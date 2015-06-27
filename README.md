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
##### 2. Start caddydev.
```shell
$ caddydev -source github.com/abiosoft/hello-caddy hello
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
$ caddydev -help
Usage: caddydev [options] directive [caddy flags]

options:
  -s, -source="."   Source code directory or go get path.
  -a, -after=""     Priority. After which directive should our new directive be placed.
  -h, -help=false   Show this usage.

directive:
  directive of the middleware being developed.

caddy flags:
  flags to pass to the resulting custom caddy binary.
```

### Note
caddydev is in active development and can still change significantly.

### License
This program is copyrighted, proprietary property and, as such, no license is granted for commercial use or redistribution.

### Disclaimer
This software is provided as-is and you assume all risk.