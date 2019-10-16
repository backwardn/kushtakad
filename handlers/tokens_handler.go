package handlers

import (
	"fmt"
	"net/http"

	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/tokens/docx"
	"github.com/kushtaka/kushtakad/tokens/pdf"
)

func GetTokens(w http.ResponseWriter, r *http.Request) {
	redirUrl := "/kushtaka/dashboard/page/1/limit/1000"
	app, err := state.Restore(r)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, "/404", 404)
		return
	}

	var tokens []models.Token
	err = app.DB.All(&tokens)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	var teams []models.Team
	err = app.DB.All(&teams)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	app.View.Teams = teams
	app.View.Tokens = tokens
	app.View.AddCrumb("Tokens", "#")
	app.View.Links.Tokens = "active"
	app.Render.HTML(w, http.StatusOK, "admin/pages/tokens", app.View)
	return
}

func PostTokens(w http.ResponseWriter, r *http.Request) {
	redirUrl := "/kushtaka/tokens/page/1/limit/100"
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	token := &models.Token{
		Name: r.FormValue("name"),
		Note: r.FormValue("note"),
		Type: r.FormValue("type"),
	}
	app.View.Forms.Token = token
	err = token.ValidateCreate()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}
	defer tx.Rollback()

	t := &models.Token{}
	tx.One("Name", token.Name, t)
	if token.ID > 0 {
		app.Fail("Token using that name already exists.")
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	switch token.Type {
	case "link":
	case "pdf":
		pdfBytes, err := app.Box.Find("files/template.pdf")
		if err != nil {
			app.Fail("Unable to find template pdf.")
			http.Redirect(w, r, redirUrl, 302)
			return
		}

		pdfCtx, err := pdf.BuildPdf(app.Settings.URI, pdfBytes)
		if err != nil {
			app.Fail("Unable to build pdf from template.")
			http.Redirect(w, r, redirUrl, 302)
			return
		}

		token.Key = pdfCtx.Key
		token.TokenContext = pdfCtx
		log.Debug(pdfCtx)
	case "docx":
		docxBytes, err := app.Box.Find("files/template.docx")
		if err != nil {
			app.Fail("Unable to find template docx.")
			http.Redirect(w, r, redirUrl, 302)
			return
		}

		docxctx, err := docx.BuildDocx(app.Settings.URI, docxBytes)
		if err != nil {
			app.Fail("Unable to build docx from template.")
			http.Redirect(w, r, redirUrl, 302)
			return
		}

		token.Key = docxctx.Key
		token.TokenContext = docxctx
		log.Debug(docxctx)
	}

	err = tx.Save(token)
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	err = tx.Commit()
	if err != nil {
		app.Fail(err.Error())
		http.Redirect(w, r, redirUrl, 302)
		return
	}

	app.View.Forms = state.NewForms()
	app.Success(fmt.Sprintf("The token [%s] was created successfully.", token.Name))
	http.Redirect(w, r, redirUrl, 302)
	return
}
