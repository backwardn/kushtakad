package server

import (
	"net/http"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/handlers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/urfave/negroni"
)

func RunServer(r chan bool, l chan models.LE) (*http.Server, *http.Server) {
	settings, n, db := handlers.ConfigureServer(r, l)
	return run(settings, n, db)
}

func run(settings *models.Settings, n *negroni.Negroni, db *storm.DB) (*http.Server, *http.Server) {
	if settings.LeEnabled {
		return HTTPS(settings, n, db)
	}
	return HTTP(settings, n), nil
}
