package server

import (
	"context"
	"net/http"
	"os"

	"github.com/kushtaka/kushtakad/angel"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

type ServerAngel struct {
	AngelCtx    context.Context
	AngelCancel context.CancelFunc

	HttpServerCtx    context.Context
	HttpServerCancel context.CancelFunc

	HttpsServerCtx    context.Context
	HttpsServerCancel context.CancelFunc

	HttpServer  *http.Server
	HttpsServer *http.Server

	Reboot chan bool
	LE     chan models.LE
}

func NewServers(sa *ServerAngel) (*http.Server, *http.Server) {
	httpctx, httpcancel := context.WithCancel(context.Background())
	httpsctx, httpscancel := context.WithCancel(context.Background())
	httpServer, httpsServer := RunServer(sa.Reboot, sa.LE)
	sa.HttpServerCtx = httpctx
	sa.HttpServerCancel = httpcancel
	sa.HttpsServerCtx = httpsctx
	sa.HttpsServerCancel = httpscancel
	return httpServer, httpsServer
}

func NewServerAngel() *ServerAngel {
	reboot := make(chan bool)
	le := make(chan models.LE)
	actx, acancel := context.WithCancel(context.Background())
	httpctx, httpcancel := context.WithCancel(context.Background())
	httpsctx, httpscancel := context.WithCancel(context.Background())
	httpServer, httpsServer := RunServer(reboot, le)
	angel.Interuptor(acancel)
	return &ServerAngel{
		AngelCtx:          actx,
		AngelCancel:       acancel,
		HttpServerCtx:     httpctx,
		HttpServerCancel:  httpcancel,
		HttpsServerCtx:    httpsctx,
		HttpsServerCancel: httpscancel,
		HttpServer:        httpServer,
		HttpsServer:       httpsServer,
		Reboot:            reboot,
		LE:                le,
	}
}

func Run() {
	sa := NewServerAngel()
	for {
		select {
		case le := <-sa.LE:
			log.Debug("Let's Encrypt Stage FQDN Test Start")

			err := le.Magic.Manage([]string{le.Domain.FQDN})
			if err != nil {
				le.Domain.LETest.State = models.LEFailed
				le.Domain.LETest.StateMsg = err.Error()
				le.DB.Update(le.Domain.LETest)
				log.Error(err)
			} else {
				le.Domain.LETest.State = models.LESuccess
				le.Domain.LETest.StateMsg = "This totally worked!"
				le.DB.Update(le.Domain.LETest)
				log.Debugf("Let's Encrypt Stage FQDN Test Successful %s", le.Domain.FQDN)
			}

			log.Debug("Let's Encrypt Stage FQDN Test End")
		case <-sa.Reboot:

			log.Debug("Begin Reboot")
			if sa.HttpServer != nil {
				log.Debug("Shutting down HTTP server")
				sa.HttpServer.Shutdown(sa.HttpServerCtx)
			}

			if sa.HttpsServer != nil {
				log.Debug("Shutting down HTTPS server")
				sa.HttpsServer.Shutdown(sa.HttpsServerCtx)
			}

			os.RemoveAll(state.AcmeTestLocation())

			sa.HttpServer, sa.HttpsServer = NewServers(sa)

			log.Info("End Reboot")
		case <-sa.AngelCtx.Done(): // if the angel's context is closed
			log.Info("shutting down Angel...done.")
			if sa.HttpServer != nil {
				sa.HttpServer.Shutdown(sa.HttpServerCtx)
			}

			if sa.HttpsServer != nil {
				sa.HttpsServer.Shutdown(sa.HttpsServerCtx)
			}

			return
		case <-sa.HttpServerCtx.Done(): // if the server's context is closed
			log.Info("shutting down HTTP ServerCtx...done.")
			return
		case <-sa.HttpsServerCtx.Done(): // if the server's context is closed
			log.Info("shutting down HTTPS ServerCtx...done.")
			return
		}
		//default:
		//https://medium.com/@ashishstiwari/dont-simply-run-forever-loop-for-1594464040b1
		// is this needed?
		// my CPU really tops out
		//time.Sleep(100 * time.Millisecond)
	}

}

/*
func le(magic certmagic.Config) error {
	// this obtains certificates or renews them if necessary
	err := magic.Manage([]string{"example.com", "sub.example.com"})
	if err != nil {
		return err
	}
}
*/
