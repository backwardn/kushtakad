package handlers

import (
	"fmt"
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetTeams(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/404", 404)
		return
	}

	var teams []models.Team
	err = app.DB.All(&teams)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Teams = teams
	app.View.Links.Teams = "active"
	app.View.AddCrumb("Teams", "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/teams", app.View)
	return
}

func PostTeams(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/teams/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	name := r.FormValue("name")
	team := &models.Team{Name: name}
	app.View.Forms.Team = team
	err = team.ValidateCreate()
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

	tx.One("Name", name, team)
	if team.ID > 0 {
		app.Fail("Team using that name already exists.")
		http.Redirect(w, r, redir, 302)
		return
	}

	err = tx.Save(team)
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

	app.View.Forms.Team = models.NewTeam()
	app.Success(fmt.Sprintf("The team [%s] was created successfully.", name))
	http.Redirect(w, r, redir, 302)
	return
}
