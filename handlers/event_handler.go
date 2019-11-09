package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

const tokenEventTmpl = `
			TokenName: %s
			<br>
			TokenType: %s
			<br>
			AttackerIP: %s
			<br>
			EventState: %s
			`

func GetTokenEvent(w http.ResponseWriter, r *http.Request) {
	log.Error("TokenEvent")
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
		log.Errorf("Seleting a token failed > %v", err)
		return
	}

	if token.ID < 1 {
		log.Errorf("token does not exist : %s", v["id"])
		return
	}

	et := &events.EventToken{
		TokenID: token.ID,
	}

	split := strings.Split(r.RemoteAddr, ":")
	ip := split[0]
	em := events.NewTokenEventManager("tcp", ip, et)
	em.AddMutex()
	em.SetState(app.DB)

	tx, err := app.DB.Begin(true)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 200, err)
		return
	}
	defer tx.Rollback()

	err = tx.Save(em)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 200, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 200, err)
		return
	}

	var team models.Team
	err = app.DB.One("ID", token.TeamID, &team)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 200, err)
		return
	}

	if em.State == "new" {
		go func() {
			e := helpers.NewEvent(app.DB, app.Box)
			e.Email.Body = fmt.Sprintf(tokenEventTmpl, token.Name, token.Type, em.AttackerIP, em.State)
			e.Email.Subject = fmt.Sprintf("ID:%d - Kushtaka Event Detected", em.ID)
			e.Email.To = team.Members
			e.Email.Filename = "event.tmpl"
			e.Email.TemplateName = "Event"
			err := e.SendEvent()
			if err != nil {
				log.Errorf("GetTokenEvent appeared to fail > %v", err)
			}
		}()
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(i)))
	http.ServeContent(w, r, "i.png", time.Now(), bytes.NewReader(i))
}
