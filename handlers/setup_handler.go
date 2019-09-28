package handlers

import (
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetSetup(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		log.Errorf("Unable to restore app %w", err)
	}

	if app.View.State.AdminIsSetup {
		app.Fail("This application already has an admin user.")
		http.Redirect(w, r, "/login", 302)
		return
	}

	var users models.User
	err = app.DB.One("ID", 1, &users)
	if err != nil {
		log.Error(err)
	}

	/*
		if r.URL.Path == "/setup" && len(users) > 0 {
			http.Redirect(w, r, "/404", 302)
			return
		}
	*/

	ren := state.NewRender("admin/layouts/center", app.Box)
	ren.HTML(w, http.StatusOK, "admin/pages/setup", app.View)
}

func PostSetup(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	if app.View.State.AdminIsSetup {
		app.Fail("This application already has an admin user.")
		http.Redirect(w, r, "/login", 302)
		return
	}

	user := &models.User{
		Email:           r.FormValue("email"),
		Password:        r.FormValue("password"),
		PasswordConfirm: r.FormValue("password_confirm"),
	}

	err = user.ValidateSetup()
	app.View.Forms.Setup = user
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	user.HashPassword()

	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	err = tx.Save(user)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	// create the default team
	team := models.NewTeam()
	team.Name = models.DefaultTeam
	team.Members = append(team.Members, user.Email)
	err = tx.Save(team)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	err = tx.Commit()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	app.Success("Admin user created successfully, please login.")
	http.Redirect(w, r, "/login", 302)
	return
}
