package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/go-oidc"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app        = kingpin.New("jwt-auth-subreq", "provides an authorization service for nginx validating JWT through subrequests")
	authDomain = app.Flag("auth-domain", "the Cloudflare auth domain to request certs from").Envar("AUTH_DOMAIN").Required().String()
	audience   = app.Flag("audience", "the expected audience of the JWT token").Envar("AUDIENCE").Required().String()
	addrIP     = app.Flag("address", "address to listen for requests on").Default("::").ResolvedIP()
	port       = app.Flag("port", "port to listen on").Default("3000").Int16()
	debug      = app.Flag("debug", "enable logging of requests").Envar("DEBUG").Default("false").Bool()
)

func certsURL(authDomain string) string {
	return fmt.Sprintf("%s/cdn-cgi/access/certs", authDomain)
}

type Middleware func(http.Handler) http.Handler

func VerifyToken(v *oidc.IDTokenVerifier) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header

			accessToken := hdr.Get("Cf-Access-Jwt-Assertion")
			if accessToken == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("missing access token on request"))
				return
			}

			ctx := r.Context()
			_, err := v.Verify(ctx, accessToken)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(fmt.Sprintf("Invalid token: %s", err.Error())))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Debug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Printf("could not dump request: %s", err.Error())
			return
		}
		log.Printf("Got request:\n%s", string(b))
	})
}

func Rescue(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("recovered from error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func compose(mw ...Middleware) Middleware {
	if len(mw) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		f := next
		for i := len(mw) - 1; i >= 0; i-- {
			f = mw[i](f)
		}
		return f
	}
}

func NoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if !strings.HasPrefix("https://", *authDomain) {
		*authDomain = "https://" + *authDomain
	}

	c := &oidc.Config{
		ClientID: *audience,
	}
	keySet := oidc.NewRemoteKeySet(context.TODO(), certsURL(*authDomain))
	verifier := oidc.NewVerifier(*authDomain, keySet, c)

	var middleware Middleware
	if !*debug {
		middleware = compose(Rescue, VerifyToken(verifier))
	} else {
		middleware = compose(Rescue, Debug, VerifyToken(verifier))
	}

	http.Handle("/", middleware(http.HandlerFunc(NoContent)))
	addr := net.JoinHostPort(addrIP.String(), strconv.Itoa(int(*port)))
	log.Printf("listening on %s", addr)
	log.Println(http.ListenAndServe(addr, nil))
}
