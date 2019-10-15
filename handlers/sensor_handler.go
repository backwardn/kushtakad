package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetSensor(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/sensors/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])

	sensor := &models.Sensor{}
	err = app.DB.One("ID", id, sensor)
	if err != nil {
		log.Error(err)
		app.Fail("Sensor does not exist")
		http.Redirect(w, r, redir, 302)
		return
	}

	for _, v := range sensor.Cfgs {
		app.View.SensorServices = append(app.View.SensorServices, v)
	}
	app.View.Sensor = sensor

	var teams []models.Team
	err = app.DB.All(&teams)
	if err != nil {
		log.Error(err)
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	var team models.Team
	err = app.DB.One("ID", sensor.TeamID, &team)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	var clones []models.Clone
	err = app.DB.All(&clones)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Clones = clones
	app.View.Team = &team
	app.View.Teams = teams
	app.View.Links.Sensors = "active"
	app.View.AddCrumb("Sensors", "/kushtaka/sensors/page/1/limit/100")
	app.View.AddCrumb(sensor.Name, "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/sensor", app.View)
	return
}

func PostSensor(w http.ResponseWriter, r *http.Request) {
	log.Error("PostSensor()")
	return
}

func UpdateSensor(w http.ResponseWriter, r *http.Request) {
	log.Error("UpdateSensor()")
	return
}

func UpdateSensorsTeam(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var newsensor models.Sensor
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&newsensor)
	if err != nil {
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	if newsensor.TeamID == 0 {
		resp = NewResponse("error", "TeamID must be greater than 0", err)
		w.Write(resp.JSON())
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	var sensor models.Sensor
	err = tx.One("ID", newsensor.ID, &sensor)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Sensor id not found, does sensor exist?", err)
		w.Write(resp.JSON())
		return
	}

	// TODO - actually perform a lookup
	sensor.TeamID = newsensor.TeamID

	err = tx.Update(&sensor)
	if err != nil {
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully updated the [%s] team", sensor.Name)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}

func DeleteSensor(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var sensor models.Sensor
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&sensor)
	if err != nil {
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	err = tx.One("ID", sensor.ID, &sensor)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Sensor id not found, does sensor exist?", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.DeleteStruct(&sensor)
	if err != nil {
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the sensor [%s]", sensor.Name)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}
