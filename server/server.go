package server

import (
	"context"
	"crypto/subtle"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/asdine/storm"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kushtaka/kushtakad/handlers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/urfave/negroni"
)

const assetsFolder = "static"
const sessionName = "_kushtaka"

var (
	rtr      *mux.Router
	fss      *sessions.FilesystemStore
	db       *storm.DB
	box      *packr.Box
	settings *models.Settings
	reboot   chan bool
	le       chan models.LE
	err      error
)

func RunServer(r chan bool, l chan models.LE) (*http.Server, *http.Server) {
	gob.Register(&state.App{})
	box = packr.New(assetsFolder, "../static")
	reboot = r
	le = l

	err = state.SetupFileStructure(box)
	if err != nil {
		log.Fatalf("Failed to setup file structure : %s", err)
	}

	db, err = storm.Open(state.DbLocation())
	if err != nil {
		log.Fatalf("Failed to open database : %s", err)
	}

	err = models.Reindex(db)
	if err != nil {
		log.Fatalf("Failed to reindex db : %s", err)
	}

	// must setup the basic hashes and settings for application to function
	settings, err = models.InitSettings()
	if err != nil {
		log.Fatalf("Failed to init settings : %s", err)
	}

	fss = sessions.NewFilesystemStore(state.SessionLocation(), settings.SessionHash, settings.SessionBlock)

	// open
	rtr = mux.NewRouter()
	rtr.HandleFunc("/assets/{theme}/{dir}/{file}", handlers.Asset).Methods("GET")
	rtr.HandleFunc("/setup", handlers.GetSetup).Methods("GET")
	rtr.HandleFunc("/setup", handlers.PostSetup).Methods("POST")
	rtr.HandleFunc("/logout", handlers.PostLogout).Methods("POST")
	rtr.HandleFunc("/t/{id}/i.png", handlers.GetTokenEvent).Methods("GET")
	rtr.HandleFunc("/", handlers.IndexCheckr).Methods("GET")
	rtr.NotFoundHandler = &NotFound{}

	// login has its own middleware chain
	login := mux.NewRouter().PathPrefix("/login").Subrouter().StrictSlash(false)
	login.Use(forceSetup)
	login.HandleFunc("", handlers.GetLogin).Methods("GET")
	login.HandleFunc("", handlers.PostLogin).Methods("POST")

	api := mux.NewRouter().PathPrefix("/api/v1").Subrouter().StrictSlash(false)
	api.Use(forceSetup)
	api.Use(isAuthenticatedWithToken)
	api.HandleFunc("/config.json", handlers.GetConfig).Methods("GET")
	api.HandleFunc("/event.json", handlers.PostEvent).Methods("POST")

	// mod has its own middleware chain
	// protected, can't process unless logged in and setup is complete
	kushtaka := mux.NewRouter().PathPrefix("/kushtaka").Subrouter().StrictSlash(true)
	kushtaka.Use(forceSetup)
	kushtaka.Use(isAuthenticated)
	kushtaka.HandleFunc("/dashboard/{pid}/1/limit/{oid}", handlers.GetDashboard).Methods("GET")

	// clones
	kushtaka.HandleFunc("/clones/page/{pid}/limit/{oid}", handlers.GetClones).Methods("GET")
	kushtaka.HandleFunc("/clones", handlers.PostClones).Methods("POST")
	kushtaka.HandleFunc("/clone", handlers.DeleteClone).Methods("DELETE")

	// sensors
	kushtaka.HandleFunc("/sensors/page/{pid}/limit/{oid}", handlers.GetSensors).Methods("GET")
	kushtaka.HandleFunc("/sensors", handlers.PostSensors).Methods("POST")

	// sensor
	kushtaka.HandleFunc("/sensor/{id}", handlers.GetSensor).Methods("GET")
	kushtaka.HandleFunc("/sensor", handlers.PostSensor).Methods("POST")
	kushtaka.HandleFunc("/sensor", handlers.DeleteSensor).Methods("DELETE")

	// service
	kushtaka.HandleFunc("/service/{sensor_id}/type/{type}", handlers.PostService).Methods("POST")
	kushtaka.HandleFunc("/service", handlers.DeleteService).Methods("DELETE")
	kushtaka.HandleFunc("/service/team/update", handlers.UpdateSensorsTeam).Methods("PUT")

	// tokens
	kushtaka.HandleFunc("/tokens/page/{pid}/limit/{oid}", handlers.GetTokens).Methods("GET")
	kushtaka.HandleFunc("/tokens", handlers.PostTokens).Methods("POST")

	kushtaka.HandleFunc("/download/token/docx/{id}", handlers.DownloadDocxToken).Methods("GET")
	kushtaka.HandleFunc("/download/token/pdf/{id}", handlers.DownloadPdfToken).Methods("GET")
	// token
	kushtaka.HandleFunc("/token/{id}", handlers.GetToken).Methods("GET")
	kushtaka.HandleFunc("/token", handlers.PostToken).Methods("POST")
	kushtaka.HandleFunc("/token", handlers.PutToken).Methods("PUT")
	kushtaka.HandleFunc("/token", handlers.DeleteToken).Methods("DELETE")

	// smtp
	kushtaka.HandleFunc("/smtp", handlers.GetSmtp).Methods("GET")
	kushtaka.HandleFunc("/smtp", handlers.PostSmtp).Methods("POST")
	kushtaka.HandleFunc("/smtp/test", handlers.PostSendTestEmail).Methods("POST")

	// users
	kushtaka.HandleFunc("/users/page/{pid}/limit/{oid}", handlers.GetUsers).Methods("GET")
	kushtaka.HandleFunc("/users", handlers.PostUsers).Methods("POST")

	// user
	kushtaka.HandleFunc("/user/{id}", handlers.GetUser).Methods("GET")
	kushtaka.HandleFunc("/user/{id}", handlers.PostUser).Methods("POST")
	kushtaka.HandleFunc("/user/{id}", handlers.PutUser).Methods("PUT")
	kushtaka.HandleFunc("/user", handlers.DeleteUser).Methods("DELETE")

	// teams
	kushtaka.HandleFunc("/teams/page/{pid}/limit/{oid}", handlers.GetTeams).Methods("GET")
	kushtaka.HandleFunc("/teams", handlers.PostTeams).Methods("POST")

	// team
	kushtaka.HandleFunc("/team/{id}", handlers.GetTeam).Methods("GET")
	kushtaka.HandleFunc("/team/{id}", handlers.PostTeam).Methods("POST")
	kushtaka.HandleFunc("/team/{id}", handlers.PutTeam).Methods("PUT")
	kushtaka.HandleFunc("/team", handlers.DeleteTeam).Methods("DELETE")

	// https
	kushtaka.HandleFunc("/https", handlers.GetHttps).Methods("GET")
	kushtaka.HandleFunc("/https/test", handlers.PostTestFQDN).Methods("POST")
	kushtaka.HandleFunc("/https/reboot", handlers.PostIRebootFQDN).Methods("POST")

	// wire up sub routers
	rtr.PathPrefix("/login").Handler(negroni.New(
		negroni.Wrap(login),
	))

	rtr.PathPrefix("/api/v1").Handler(negroni.New(
		negroni.Wrap(api),
	))

	rtr.PathPrefix("/kushtaka").Handler(negroni.New(
		negroni.Wrap(kushtaka),
	))

	rtr.HandleFunc("/ws", handlers.Ws)

	// setup router
	n := negroni.New()
	n.Use(negroni.HandlerFunc(before))
	n.UseHandler(rtr)
	n.Use(negroni.HandlerFunc(after))

	return run(settings, n, db)
}

func run(settings *models.Settings, n *negroni.Negroni, db *storm.DB) (*http.Server, *http.Server) {
	if settings.LeEnabled {
		return HTTPS(settings, n, db)
	}
	return HTTP(settings, n), nil
}

// forceSetup is a middleware function that makes sure
// a admin user is created before allowing the user to
// proceed with using the application
func forceSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(state.AppStateKey).(*state.App)
		var user models.User
		err := db.One("ID", 1, &user)
		if err != nil && r.URL.Path != "/setup" {
			app.Fail("You must create an admin user before proceeding.")
			http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(state.AppStateKey).(*state.App)
		if app.User.ID < 1 {
			app.Fail("You must login before proceeding.")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthenticatedWithToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey string
		app := r.Context().Value(state.AppStateKey).(*state.App)
		token, ok := r.Header["Authorization"]
		if ok && len(token) >= 1 {
			apiKey = token[0]
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}

		var sensor models.Sensor
		app.DB.One("ApiKey", apiKey, &sensor)
		if subtle.ConstantTimeCompare([]byte(sensor.ApiKey), []byte(apiKey)) == 0 {
			app.Render.JSON(w, 401, "")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func before(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	// setup session and if it errors, create a new session
	sess, err := fss.Get(r, sessionName)
	if err != nil {
		fss.New(r, sessionName)
		sess, err = fss.Get(r, sessionName)
	}
	sess.Options.HttpOnly = true

	cfg := &state.Config{
		Reponse:         w,
		Request:         r,
		DB:              db,
		Session:         sess,
		FilesystemStore: fss,
		Box:             box,
		Reboot:          reboot,
		LE:              le,
		Settings:        settings,
	}
	app, err := state.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	app.RestoreFlash()
	app.RestoreUser()
	app.RestoreForm()
	app.RestoreState()
	app.RestoreURI()

	ctx := context.WithValue(r.Context(), state.AppStateKey, app)
	next(w, r.WithContext(ctx))
}

func after(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	app := r.Context().Value(state.AppStateKey).(*state.App)

	// because we build the view upon each request
	// we clear it here to keep consistency and state
	//

	userState, err := json.Marshal(app.User)
	if err != nil {
		log.Fatal(err)
	}

	formState, err := json.Marshal(app.View.Forms)
	if err != nil {
		log.Fatal(err)
	}

	app.Session.Values[state.UserStateKey] = userState
	app.Session.Values[state.FormStateKey] = formState
	err = app.Session.Save(r, w)
	if err != nil {
		log.Fatal(err)
	}

	app.View.Clear()

	next(w, r)
}

//
// NOT FOUND
//
type NotFound struct{}

func (nf *NotFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("404 Not Found"))
	return
}
