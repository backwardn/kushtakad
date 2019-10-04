package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetHttps(w http.ResponseWriter, r *http.Request) {
	redir := "/kushtaka/dashboard"
	app, err := state.Restore(r)
	if err != nil {
		app.Render.JSON(w, 404, err)
		return
	}

	var letests []models.LETest
	err = app.DB.All(&letests, storm.Reverse())
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}

	app.View.LETests = letests
	app.View.Links.Https = "active"
	app.View.AddCrumb("HTTPS", "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/https", app.View)
	return
}

func PostTestFQDN(w http.ResponseWriter, r *http.Request) {
	log.Debug("Start")
	app, err := state.Restore(r)
	if err != nil {
		resp := NewResponse("failed", "failed to restore", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	var domain models.Domain
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&domain)
	if err != nil {
		resp := NewResponse("failed", "FQDN not provided?", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp := NewResponse("failed", "FQDN not provided?", err)
		app.Render.JSON(w, 200, resp)
		return
	}
	defer tx.Rollback()

	var letests []models.LETest
	err = app.DB.Select(
		q.Eq("FQDN", domain.FQDN),
		q.Eq("State", models.LEPending)).Find(&letests)

	if len(letests) > 0 {
		resp := NewResponse("failed", "That FQDN is currently in a pending state", nil)
		app.Render.JSON(w, 200, resp)
		return
	}

	letest := &models.LETest{
		FQDN:    domain.FQDN,
		State:   models.LEPending,
		Created: time.Now(),
	}

	err = tx.Save(letest)
	if err != nil {
		resp := NewResponse("failed", "Failed to save the LETest struct", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("failed", "Failed to save the LETest struct", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	domain.LETest = letest
	le := models.NewStageLE(app.User.Email, state.DataDirLocation(), domain, app.DB)
	app.LE <- le

	resp := NewResponse("success", "Succes to test LETest", nil)
	app.Render.JSON(w, 200, resp)
	log.Debug("End")
	return
}

func PostIRebootFQDN(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		resp := NewResponse("failed", "failed to restore", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	var let models.LETest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&let)
	if err != nil {
		resp := NewResponse("failed", "FQDN not provided?", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	err = app.DB.One("ID", let.ID, &let)
	if err != nil {
		resp := NewResponse("failed", "Cannot find the LETest ID you provided", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	set, err := models.NewSettings()
	if err != nil {
		resp := NewResponse("failed", "Unable to init NewSettings()", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	set.LeFQDN = let.FQDN
	set.URI = let.FQDN
	set.LeEnabled = true
	err = set.WriteSettings()
	if err != nil {
		resp := NewResponse("failed", "Unable to write settings", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	err = app.DB.Close()
	if err != nil {
		resp := NewResponse("failed", "Unable to close db", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	app.Reboot <- true

	resp := NewResponse("success", "Reboot...", nil)
	app.Render.JSON(w, 200, resp)
	log.Debug("End")
	return
}

/*
app.Reboot <- true
var wg sync.WaitGroup

wg.Add(1)
go func() {
	magic := certmagic.NewDefault()
	magic.CA = certmagic.LetsEncryptStagingCA
	magic.Email = "jfolkins@gmail.com"
	magic.Agreed = true
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Debug("Lookit my cool website over HTTPS!")
		wg.Done()
	})
	err = http.ListenAndServe(":80", magic.HTTPChallengeHandler(mux))
	if err != nil {
		log.Debug(err)
	}
}()
wg.Wait()
*/
