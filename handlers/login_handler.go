package handlers

import (
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func PostLogout(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	err = app.EndSession()
	if err != nil {
		log.Debug(err)
		w.Write([]byte("{\"msg\":\"fail\"}"))
	}

	w.Write([]byte("{\"msg\":\"success\"}"))
	return
}
func GetLogin(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	if app.View.State.IsAuthd {
		app.Fail("You are already authenticated.")
		http.Redirect(w, r, redir, 302)
		return
	}

	ren := state.NewRender("admin/layouts/center", app.Box)
	ren.HTML(w, http.StatusOK, "admin/pages/login", app.View)
}

func PostLogin(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	if app.View.State.IsAuthd {
		app.Fail("You are already authenticated.")
		http.Redirect(w, r, redir, 302)
		return
	}

	extUser := &models.User{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}
	err = extUser.ValidateLogin()
	app.View.Forms.Login = extUser

	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/setup", 302)
		return
	}

	user := &models.User{}
	err = app.DB.One("Email", extUser.Email, user)
	if err != nil {
		app.Fail("User is not found.")
		http.Redirect(w, r, "/login", 302)
		return
	}

	err = user.Authenticate(extUser.Password)
	if err != nil {
		app.Fail("User or password is incorrect.")
		http.Redirect(w, r, "/login", 302)
		return
	}

	app.User = user
	app.Success("You have successfully logged in.")
	http.Redirect(w, r, redir, 302)
	return
}

func ReadSettings(w http.ResponseWriter, r *http.Request) {
	log.Error("read settings")
	return
}
