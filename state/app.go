package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/asdine/storm"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/sessions"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/unrolled/render"
)

const AppStateKey = "AppState"
const ViewStateKey = "ViewData"
const UserStateKey = "UserState"
const FormStateKey = "FormState"
const FlashFail = "FlashFail"
const FlashSuccess = "FlashSuccess"

type App struct {
	Response  http.ResponseWriter
	Request   *http.Request
	DB        *storm.DB
	Settings  *models.Settings
	Session   *sessions.Session
	FileStore *sessions.FilesystemStore
	Box       *packr.Box
	View      *View
	User      *models.User
	Render    *render.Render
	ServerHub *ServerHub
	Reboot    chan bool
	LE        chan models.LE
}

func tmplFuncs() []template.FuncMap {
	funks := []template.FuncMap{}
	var fns = template.FuncMap{
		"prettytime": helpers.PrettyTime,
		"plus1": func(x int) int {
			return x + 1
		},
		"date": func(d time.Time) string {
			dt := fmt.Sprintf("%v-%v-%v", d.Year(), d.Month(), d.Day())
			return dt
		},
		"unsafe": func(s string) string {
			return html.UnescapeString(s)
		},
	}
	funks = append(funks, fns)
	return funks
}
func NewRender(layout string, box *packr.Box) *render.Render {
	dummyDir := "__DUM__"
	return render.New(render.Options{
		Asset: func(name string) ([]byte, error) {
			name = strings.TrimPrefix(name, dummyDir)
			name = strings.TrimPrefix(name, "/")
			return box.Find(name)
		},
		AssetNames: func() []string {
			names := box.List()
			for k, v := range names {
				pth := path.Join(dummyDir, v)
				names[k] = pth
			}
			return names
		},
		Funcs:           tmplFuncs(),
		Directory:       dummyDir, // Specify what path to load the templates from.
		Extensions:      []string{".tmpl", ".html"},
		Layout:          layout, // Specify a layout template. Layouts can call {{ yield }} to render the current template or {{ partial "css" }} to render a partial from the current template.
		RequirePartials: true,   // Return an error if a template is missing a partial used in a layout.
	})
}

type Config struct {
	Reponse         http.ResponseWriter
	Request         *http.Request
	DB              *storm.DB
	Session         *sessions.Session
	Settings        *models.Settings
	FilesystemStore *sessions.FilesystemStore
	Box             *packr.Box
	Reboot          chan bool
	LE              chan models.LE
}

// NewApp returns and instance of App
// App instances live during the lifecycle of a single http request
func NewApp(cfg *Config) (*App, error) {

	ren := NewRender("admin/layouts/main", cfg.Box)

	// TODO this is for websockets? I have no idea what I was thinking but a new instance doesn't seem right
	// this should be move to server?
	//hub := newServerHub(cfg.DB)
	//go hub.run()

	return &App{
		//ServerHub: hub,
		Response:  cfg.Reponse,
		Request:   cfg.Request,
		DB:        cfg.DB,
		Session:   cfg.Session,
		FileStore: cfg.FilesystemStore,
		Box:       cfg.Box,
		Settings:  cfg.Settings,
		Reboot:    cfg.Reboot,
		LE:        cfg.LE,
		Render:    ren,
		View:      NewView(),
		User:      &models.User{},
	}, nil

}

func (app *App) NotFound(msg string, err error) {
	ren := render.New(render.Options{
		Extensions:      []string{".tmpl", ".html"},
		Directory:       "static",               // Specify what path to load the templates from.
		Layout:          "admin/layouts/center", // Specify a layout template. Layouts can call {{ yield }} to render the current template or {{ partial "css" }} to render a partial from the current template.
		RequirePartials: true,                   // Return an error if a template is missing a partial used in a layout.
	})
	ren.HTML(app.Response, http.StatusNotFound, "admin/pages/404", app.View)
}

func (app *App) RestoreForm() {
	val := app.Session.Values[FormStateKey]

	if b, ok := val.([]byte); ok {
		var forms = NewForms()
		err := json.Unmarshal(b, forms)
		if err != nil {
			panic(err)
		}

		app.View.Forms = forms
	}
}

func (app *App) RestoreUser() {
	val := app.Session.Values[UserStateKey]

	if b, ok := val.([]byte); ok {
		var us = models.NewUser()
		err := json.Unmarshal(b, us)
		if err != nil {
			panic(err)
		}

		app.User = us
		app.View.User = us
	}
}

func (app *App) RestoreURI() {
	app.View.URI = app.Settings.BuildURI()
}

func (app *App) RestoreState() {
	st := models.NewState(app.User, app.DB)
	app.View.State = st
}

func (v *View) Clear() {
	v = &View{}
}

func Restore(r *http.Request) (*App, error) {
	app := r.Context().Value(AppStateKey).(*App)
	if app == nil {
		return nil, errors.New("Application unable to restore restore")
	}
	return app, nil
}

func (app *App) RestoreFlash() {
	for _, fe := range app.Session.Flashes(FlashFail) {
		app.View.FlashFail = append(app.View.FlashFail, fe.(string))
	}

	for _, fe := range app.Session.Flashes(FlashSuccess) {
		app.View.FlashSuccess = append(app.View.FlashSuccess, fe.(string))
	}
}

func (app *App) Fail(msg string) {
	s := strings.Split(msg, ";")
	if len(s) > 0 {
		for _, v := range s {
			app.Session.AddFlash(v, FlashFail)
		}
	} else {
		app.Session.AddFlash(msg, FlashFail)
	}
	app.Session.Save(app.Request, app.Response)
}

func (app *App) Success(msg string) {
	app.Session.AddFlash(msg, FlashSuccess)
	app.Session.Save(app.Request, app.Response)
}

func (app *App) EndSession() error {
	app.User = &models.User{}
	app.View = &View{}
	err := app.Session.Save(app.Request, app.Response)
	if err != nil {
		return err
	}

	return nil
}
