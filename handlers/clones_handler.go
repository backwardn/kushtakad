package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/kushtaka/kushtakad/clone"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/storage"
)

func GetClones(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatalf("App failed to restored: %s", err.Error())
		app.Fail(err.Error())
		http.Redirect(w, r, "/404", 404)
		return
	}
	app.View.Links.Clones = "active"

	var clones []models.Clone
	err = app.DB.All(&clones)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	app.View.Clones = clones

	app.View.AddCrumb("Clones", "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/clones", app.View)
	return
}

func PostClones(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/clones/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	ffqdn := r.FormValue("fqdn")
	fqdn, err := url.ParseRequestURI(ffqdn)
	if err != nil {
		msg := fmt.Sprintf("The fqdn has issues > %s", err.Error())
		app.Fail(msg)
		http.Redirect(w, r, redir, 302)
		return
	}
	log.Debug(ffqdn)

	//go func() {
	db, err := storage.MustDBWithLocationAndName(state.ClonesLocation(), fqdn.Hostname())
	if err != nil {
		log.Error(err)
		return
	}
	defer db.Close()

	err = clone.Run(ffqdn, 2, db)
	if err != nil {
		log.Error(err)
		return
	}
	//}()

	mclone := &models.Clone{}
	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	defer tx.Rollback()

	sc := models.NewClone()
	sc.FQDN = fqdn.Scheme + "://" + fqdn.Hostname() + fqdn.Port()
	sc.Depth = 1
	tx.One("FQDN", fqdn, &mclone)
	if mclone.ID > 0 {
		err = tx.Update(sc)
		if err != nil {
			msg := fmt.Sprintf("tx.Update() failed > %s", err.Error())
			app.Fail(msg)
			http.Redirect(w, r, redir, 302)
			return
		}
	} else {
		err = tx.Save(sc)
		if err != nil {
			msg := fmt.Sprintf("tx.Save() failed > %s", err.Error())
			app.Fail(msg)
			http.Redirect(w, r, redir, 302)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.Forms = state.NewForms()
	//app.Success(fmt.Sprintf("The clone [%s] was created successfully.", sc.FQDN))
	app.Success(fmt.Sprintf("The clone [%s] was created successfully.", "NOT REALLY"))
	http.Redirect(w, r, redir, 302)
	return
}
