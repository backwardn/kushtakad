package handlers

import (
	"net/http"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/state"
)

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	base := "/kushtaka/dashboard"
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	var events []events.EventManager
	app.DB.All(&events, storm.Reverse())

	app.View.Pagi.BaseURI = base
	app.View.Pagi.Configure(len(events), r)

	app.DB.All(&events, storm.Limit(app.View.Pagi.Limit), storm.Skip(app.View.Pagi.Offset), storm.Reverse())
	log.Debugf("total events found %d", len(events))

	app.View.AddCrumb("Dashboard", "#")
	app.View.Events = events
	app.View.Links.Dashboard = "active"
	app.Render.HTML(w, http.StatusOK, "admin/pages/dashboard", app.View)
	return
}
