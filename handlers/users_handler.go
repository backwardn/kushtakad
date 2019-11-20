package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetUser(w http.ResponseWriter, r *http.Request) {
	log.Error("GetUser()")
	return
}

func PostUser(w http.ResponseWriter, r *http.Request) {
	log.Error("PostUser()")
	return
}

func PutUser(w http.ResponseWriter, r *http.Request) {
	log.Error("PutUser()")
	return
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var user models.User
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&user)
	if err != nil {
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	err = user.Delete(app.DB)
	if err != nil {
		resp = NewResponse("error", "Unable to delete the user", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the user [%s]", user.Email)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	redirUrl := "/kushtaka/users/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/404", 404)
		return
	}

	var users []models.User
	err = app.DB.All(&users)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	app.View.Users = users
	app.View.AddCrumb("Users", "#")
	app.View.Links.Users = "active"
	app.Render.HTML(w, http.StatusOK, "admin/pages/users", app.View)
	return
}

func PostUsers(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/users/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
	}

	user := &models.User{
		Email:           r.FormValue("email"),
		Password:        r.FormValue("password"),
		PasswordConfirm: r.FormValue("password_confirm"),
	}

	err = user.Create(db)
	app.View.Forms.User = user
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Forms = state.NewForms()
	app.Success("User created successfully")
	http.Redirect(w, r, redir, 302)
	return
}
