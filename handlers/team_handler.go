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

func GetTeam(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/teams/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	team := &models.Team{}
	err = app.DB.One("ID", id, team)
	if err != nil {
		app.Fail("Team does not exist")
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Team = team

	app.View.Links.Teams = "active"
	app.View.AddCrumb("Teams", redir)
	app.View.AddCrumb(team.Name, "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/team", app.View)
	return
}

func PostTeam(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	email := r.FormValue("email")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		app.Fail("Unable to parse ID")
		http.Redirect(w, r, "/kushtaka/teams/page/1/limit/100", 302)
		return
	}

	team := &models.Team{}
	err = app.DB.One("ID", id, team)
	if err != nil {
		app.Fail("Team does not exist. " + err.Error())
		http.Redirect(w, r, "/kushtaka/teams/page/1/limit/100", 302)
		return
	}

	url := "/kushtaka/team/" + vars["id"]
	err = team.ValidateAddMember(email)
	app.View.Forms.TeamMember = team
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, url, 302)
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, url, 302)
		return
	}
	team.MemberToAdd = ""

	err = tx.Save(team)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, url, 302)
		return
	}

	err = tx.Commit()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/kushtaka/dashboard", 302)
		return
	}

	app.View.Forms = state.NewForms()
	app.Success("Member has been successfully added to the team.")
	http.Redirect(w, r, url, 302)
	return
}

func PutTeam(w http.ResponseWriter, r *http.Request) {
	log.Error("PutTeam()")
	return
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var team models.Team
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&team)
	if err != nil {
		log.Error(err)
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		log.Error(err)
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	err = tx.One("ID", team.ID, &team)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Team id not found, does team exist?", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.DeleteStruct(&team)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the team [%s]", team.Name)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}

func DeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Error(err)
		resp = NewResponse("error", "Can't convert Team ID to Int", err)
		w.Write(resp.JSON())
		return
	}
	log.Debugf("Team ID is %d", id)

	tx, err := app.DB.Begin(true)
	if err != nil {
		log.Error(err)
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	var team models.Team
	err = tx.One("ID", id, &team)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Team id not found, does team exist?", err)
		w.Write(resp.JSON())
		return
	}

	var user models.User
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&user)
	if err != nil {
		log.Error(err)
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	tx.One("Email", user.Email, &user)
	if user.ID == 1 && team.Name == "Default" {
		resp := NewResponse("error", "Unable to delete the default user from the default team", nil)
		w.Write(resp.JSON())
		return
	}

	for i, v := range team.Members {
		if v == user.Email {
			team.Members = append(team.Members[:i], team.Members[i+1:]...)
			break
		}
	}

	err = tx.Update(&team)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the team [%s]", team.Name)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}
