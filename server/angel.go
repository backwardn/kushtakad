package server

import (
	"context"
	"net/http"

	"github.com/kushtaka/kushtakad/angel"
	"github.com/kushtaka/kushtakad/models"
)

type ServerAngel struct {
	AngelCtx     context.Context
	AngelCancel  context.CancelFunc
	ServerCtx    context.Context
	ServerCancel context.CancelFunc
	Server       *http.Server
	Reboot       chan bool
	LE           chan models.LE
}

func NewServer(sa *ServerAngel) *http.Server {
	sctx, scancel := context.WithCancel(context.Background())
	srv := RunServer(sa.Reboot, sa.LE)
	sa.ServerCtx = sctx
	sa.ServerCancel = scancel
	return srv
}

func NewServerAngel() *ServerAngel {
	reboot := make(chan bool)
	le := make(chan models.LE)
	actx, acancel := context.WithCancel(context.Background())
	sctx, scancel := context.WithCancel(context.Background())
	srv := RunServer(reboot, le)
	angel.Interuptor(acancel)
	return &ServerAngel{
		AngelCtx:     actx,
		AngelCancel:  acancel,
		ServerCtx:    sctx,
		ServerCancel: scancel,
		Server:       srv,
		Reboot:       reboot,
		LE:           le,
	}
}

func Run() {
	sa := NewServerAngel()
	for {
		select {
		case le := <-sa.LE:
			log.Debug("Let's Encrypt Start")

			err := le.Magic.Manage([]string{le.Domain.FQDN})
			if err != nil {
				le.Domain.LETest.State = models.LEFailed
				le.Domain.LETest.StateMsg = err.Error()
				le.DB.Update(le.Domain.LETest)
				log.Fatal(err)
			} else {
				le.Domain.LETest.State = models.LESuccess
				le.Domain.LETest.StateMsg = "This totally worked!"
				le.DB.Update(le.Domain.LETest)
				log.Debugf("Let's Encrypt Successful %s", le.Domain.FQDN)
			}

			log.Debug("Let's Encrypt End")
		case <-sa.Reboot:
			log.Info("Reboot channel signaled...")
			log.Info("Shutting down server...")
			sa.Server.Shutdown(sa.ServerCtx)
			log.Info("Done.")
			log.Info("Rebooting server...")
			sa.Server = NewServer(sa)
			log.Info("Done.")
		case <-sa.AngelCtx.Done(): // if the angel's context is closed
			log.Info("shutting down Angel...done.")
			sa.Server.Shutdown(sa.ServerCtx)
			return
		case <-sa.ServerCtx.Done(): // if the server's context is closed
			log.Info("shutting down ServerCtx...done.")
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
