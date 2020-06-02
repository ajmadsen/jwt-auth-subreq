# jwt-auth-subreq

Authorizes JWT for nginx through subrequests

A quick and sloppy service I whipped up adapting [Cloudflare's Example](https://developers.cloudflare.com/access/setting-up-access/validate-jwt-tokens/)
for my own personal use.

```text
usage: jwt-auth-subreq --auth-domain=AUTH-DOMAIN --audience=AUDIENCE [<flags>]


Flags:
  --help                     Show context-sensitive help (also try --help-long
                             and --help-man).
  --auth-domain=AUTH-DOMAIN  the Cloudflare auth domain to request certs from
  --audience=AUDIENCE        the expected audience of the JWT token
  --address=::               address to listen for requests on
  --port=3000                port to listen on
  --debug                    enable logging of requests
```

Usage with nginx:

```conf
server {
  ...

  location / {
    auth_request /_auth;
    ...
  }

  location = /_auth {
    internal;
    proxy_pass http://localhost:3000;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
  }

  ...
}
```
