package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

func GetHttps(w http.ResponseWriter, r *http.Request) {
	app, err := state.Restore(r)
	if err != nil {
		app.Render.JSON(w, 404, err)
		return
	}

	app.View.Links.Https = "active"
	app.View.AddCrumb("HTTPS", "#")
	app.Render.HTML(w, http.StatusOK, "admin/pages/https", app.View)
	return
}

type Domain struct {
	FQDN string `json:"fqdn"`
}

func PostTestFQDN(w http.ResponseWriter, r *http.Request) {
	log.Debug("Start")
	app, err := state.Restore(r)
	if err != nil {
		resp := NewResponse("failed", "failed to restore", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	var domain Domain
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&domain)
	if err != nil {
		resp := NewResponse("failed", "FQDN not provided?", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	fqdn := models.NewFQDN()
	fqdn.Test(domain.FQDN)
	var resps []*Response
	if !fqdn.Port80.Test {
		resp := NewResponse("failed", "Binding to port :80 failed", fqdn.Port80.Err)
		resp.Type = "port-80-answer"
		resp.Obj = fqdn.Port80
		resps = append(resps, resp)
	} else {
		resp := NewResponse("success", "Binding to port :80 succeeded", nil)
		resp.Type = "port-80-answer"
		resp.Obj = fqdn.Port80
		resps = append(resps, resp)
	}

	if !fqdn.Port443.Test {
		resp := NewResponse("failed", "Binding to port :443 failed", fqdn.Port443.Err)
		resp.Type = "port-443-answer"
		resp.Obj = fqdn.Port443
		resps = append(resps, resp)
	} else {
		resp := NewResponse("success", "Binding to port :443 succeeded", nil)
		resp.Type = "port-443-answer"
		resp.Obj = fqdn.Port443
		resps = append(resps, resp)
	}

	if !fqdn.IPMatch.Test {
		resp := NewResponse("failed", "Outbound IP address doesn't match your server's", fqdn.IPMatch.Err)
		resp.Type = "ip-match-answer"
		resp.Obj = fqdn.IPMatch
		resps = append(resps, resp)
	} else {
		resp := NewResponse("success", "Outbound IP address matches", nil)
		resp.Type = "ip-match-answer"
		resp.Obj = fqdn.IPMatch
		resps = append(resps, resp)
	}

	if !fqdn.ARecord.Test {
		resp := NewResponse("failed", "(a) record IP doesn't match your server's IP", fqdn.ARecord.Err)
		resp.Type = "a-record-answer"
		resp.Obj = fqdn.ARecord
		resps = append(resps, resp)
	} else {
		resp := NewResponse("success", "(a) record IP matches server's IP", nil)
		resp.Type = "a-record-answer"
		resp.Obj = fqdn.ARecord
		resps = append(resps, resp)
	}

	//le := models.NewStageLE(app.User.Email, []string{domain.FQDN})
	//app.LE <- le

	app.Render.JSON(w, 200, resps)
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

	var domain Domain
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&domain)
	if err != nil {
		resp := NewResponse("failed", "FQDN not provided?", err)
		app.Render.JSON(w, 200, resp)
		return
	}

	var resps []*Response
	le := models.NewStageLE(app.User.Email, []string{domain.FQDN})
	app.LE <- le
	resp := NewResponse("success", "Outbound IP address matches", nil)
	resp.Type = "ip-match-answer"
	resp.Obj = le
	resps = append(resps, resp)
	app.Render.JSON(w, 200, resps)
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
