package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/asdine/storm"
	"github.com/pkg/browser"
	"github.com/urfave/negroni"
)

func HTTPS(settings *models.Settings, mux http.Handler, db *storm.DB) (*http.Server, *http.Server) {
	var httpLn net.Listener
	var httpsLn net.Listener
	var httpWg sync.WaitGroup
	var lnMu sync.Mutex
	var err error


	domain := models.Domain{FQDN: settings.LeFQDN}
	le := models.NewLE(state.DataDirLocation(), domain, db)
	cfg := le.Magic
	err = cfg.Manage([]string{settings.LeFQDN})
	if err != nil {
		log.Error(err)
		return nil, nil
	}

	httpWg.Add(1)
	defer httpWg.Done()

	// if we haven't made listeners yet, do so now,
	// and clean them up when all servers are done
	lnMu.Lock()
	httpLn, err = net.Listen("tcp", fmt.Sprintf(":%d", 80))
	if err != nil {
		lnMu.Unlock()
		log.Error(err)
		return nil, nil
	}

	httpsLn, err = tls.Listen("tcp", fmt.Sprintf(":%d", 443), cfg.TLSConfig())
	if err != nil {
		httpLn.Close()
		httpLn = nil
		lnMu.Unlock()
		log.Error(err)
		return nil, nil
	}

	hln, hsln := httpLn, httpsLn
	lnMu.Unlock()

	// create HTTP/S servers that are configured
	// with sane default timeouts and appropriate
	// handlers (the HTTP server solves the HTTP
	// challenge and issues redirects to HTTPS,
	// while the HTTPS server simply serves the
	// user's handler)
	httpServer := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
		Handler:           cfg.HTTPChallengeHandler(http.HandlerFunc(httpRedirectHandler)),
	}
	httpsServer := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       5 * time.Minute,
		Handler:           mux,
	}

	log.Debugf("%v Serving HTTP->HTTPS on %s and %s",
		settings.LeFQDN, hln.Addr(), hsln.Addr())

	go httpServer.Serve(hln)
	go httpsServer.Serve(hsln)
	return httpServer, httpsServer
}

func HTTP(settings *models.Settings, n *negroni.Negroni) *http.Server {
	env := os.Getenv("KUSHTAKA_ENV")

	go func() {
		time.Sleep(1 * time.Second)
		log.Infof("Listening on...%s\n", settings.Host)
		if env != "development" {
			err := browser.OpenURL(settings.URI)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	log.Debugf("settings.Host %s", settings.Host)
	log.Debugf("settings.URI %s", settings.URI)

	srv := &http.Server{Addr: settings.Port, Handler: n}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("The http server died :%s", err)
		}
	}()
	return srv
}

func httpRedirectHandler(w http.ResponseWriter, r *http.Request) {
	toURL := "https://"

	// since we redirect to the standard HTTPS port, we
	// do not need to include it in the redirect URL
	requestHost, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		requestHost = r.Host // host probably did not contain a port
	}

	toURL += requestHost
	toURL += r.URL.RequestURI()

	// get rid of this disgusting unencrypted HTTP connection 🤢
	w.Header().Set("Connection", "close")

	http.Redirect(w, r, toURL, http.StatusMovedPermanently)
}
