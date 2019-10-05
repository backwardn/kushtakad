package handlers

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetTokenEvent(w http.ResponseWriter, r *http.Request) {
	log.Error("test token")
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
		return
	}

	v := mux.Vars(r)
	i, err := app.Box.Find("files/i.png")
	if err != nil {
		log.Error(err)
		return
	}

	var token models.Token
	err = app.DB.One("Key", v["id"], &token)
	if err != nil {
		log.Error(err)
		return
	}

	if token.ID < 1 {
		log.Errorf("token does not exist : %s", v["id"])
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(i)))
	http.ServeContent(w, r, "i.png", time.Now(), bytes.NewReader(i))
}
