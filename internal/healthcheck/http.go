package healthcheck

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

// ListenAndServeTLS is a drop-in replacement for a default
// ListenAndServeTLS func from the http std library. The func
// takes not paths to a cert and a key but slices of bytes representing
// content of the files used by the original http.ListenAndServeTLS function.
func ListenAndServeTLS(addr string, cert, key []byte, handler http.Handler) error {
	tlsCert, err := LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServeTLS("", "")
}

// LoadX509KeyPair is a drop-in replacement for the LoadX509KeyPair
// from the http package from the std library. It takes not files
// containing a cert and a key but a slice of bytes representing
// the content of the corresponding files.
func LoadX509KeyPair(cert, key []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}

// API constructs an http.Handler with all healtcheck routes.
func API(client *client.NginxClient, cnf *configs.Configurator) http.Handler {
	health := HealthHandler{
		client: client,
		cnf:    cnf,
	}
	mux := chi.NewRouter()
	mux.MethodFunc(http.MethodGet, "/", health.Info)
	mux.MethodFunc(http.MethodGet, "/healthcheck/{hostname}", health.Retrieve)
	return mux
}

// RunHealtcheckServer takes configs and starts healtcheck service.
func RunHealtcheckServer(port string, nc *client.NginxClient, cnf *configs.Configurator) {
	healthServer := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: API(nc, cnf),

		// For now hardcoded!
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	healthServer.ListenAndServe()
}
