package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/kushtaka/kushtakad/clone"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/storage"
)

func DeleteClone(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var clone models.Clone
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&clone)
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

	err = tx.One("ID", clone.ID, &clone)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Clone id not found, does clone exist?", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.DeleteStruct(&clone)
	if err != nil {
		resp := NewResponse("error", "Unable to update clone", err)
		w.Write(resp.JSON())
		return
	}

	fp := path.Join(state.ServerClonesLocation(), clone.Hostname)
	err = os.Remove(fp)
	if err != nil {
		resp := NewResponse("error", "Unable to delete cloned db", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the clone [%s]", clone.FQDN)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}

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

	depth, err := strconv.Atoi(r.FormValue("http-depth"))
	if err != nil {
		msg := fmt.Sprintf("The depth param has issues > %s", err.Error())
		app.Fail(msg)
		http.Redirect(w, r, redir, 302)
		return
	}

	log.Debug(depth)
	log.Debug(fqdn.Hostname())

	//go func() {
	db, err := storage.MustDBWithLocationAndName(state.ServerClonesLocation(), fqdn.Hostname())
	if err != nil {
		log.Error(err)
		return
	}
	defer db.Close()

	err = clone.Run(ffqdn, depth, db)
	if err != nil {
		msg := fmt.Sprintf("Unable to run the clone process > %s", err.Error())
		log.Error(msg)
		app.Fail(msg)
		http.Redirect(w, r, redir, 302)
		return
	}
	//}()

	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redir, 302)
		return
	}
	defer tx.Rollback()

	mclone := &models.Clone{}
	sc := models.NewClone()
	sc.Hostname = fqdn.Hostname()
	// TODO: horrible, fix soon
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

	// make root / if one does not exist
	var res clone.Res
	var rd clone.Redirect
	db.One("URL", "/", &res)
	db.One("URL", "/", &rd)

	if res.ID == 0 && rd.ID == 0 {
		newrd := &clone.Redirect{
			URL:        "/",
			StatusCode: 302,
			GotoURL:    fqdn.RequestURI(),
		}
		err := db.Save(newrd)
		if err != nil {
			log.Errorf("Unable to save missing Redirect in clone > %v", err)
			app.Fail(err.Error())
			http.Redirect(w, r, redir, 302)
			return
		}
	}

	app.View.Forms = state.NewForms()
	app.Success(fmt.Sprintf("The clone [%s] was created successfully.", sc.FQDN))
	http.Redirect(w, r, redir, 302)
	return
}
