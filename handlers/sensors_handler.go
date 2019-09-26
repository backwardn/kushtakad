package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetSensors(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatalf("App failed to restored: %s", err.Error())
		app.Fail(err.Error())
		http.Redirect(w, r, "/404", 404)
		return
	}
	app.View.Links.Sensors = "active"

	var sensors []models.Sensor
	err = app.DB.All(&sensors)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	app.View.Sensors = sensors

	var teams []models.Team
	err = app.DB.All(&teams)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	app.View.Teams = teams
	app.View.AddCrumb("Sensors", "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/sensors", app.View)
	return
}

func PostSensors(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/sensors/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	name := r.FormValue("name")
	note := r.FormValue("note")
	log.Debug(name)
	tid, err := strconv.ParseInt(r.FormValue("teamId"), 10, 64)
	if err != nil {
		app.Fail("Please select a team to notify")
		http.Redirect(w, r, redir, 302)
		return
	}

	sensor := models.NewSensor(name, note, tid)
	app.View.Forms.Sensor = sensor
	err = sensor.ValidateCreate()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	defer tx.Rollback()

	tx.One("Name", name, &sensor)
	if sensor.ID > 0 {
		app.Fail("Sensor using that name already exists.")
		http.Redirect(w, r, redir, 302)
		return
	}

	err = tx.Save(sensor)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	err = tx.Commit()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Forms = state.NewForms()
	app.Success(fmt.Sprintf("The sensor [%s] was created successfully.", sensor.Name))
	http.Redirect(w, r, redir, 302)
	return
}
