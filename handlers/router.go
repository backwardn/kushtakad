package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/gob"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/asdine/storm"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
	"github.com/urfave/negroni"
)

var (
	fss      *sessions.FilesystemStore
	db       *storm.DB
	box      *packr.Box
	settings *models.Settings
	reboot   chan bool
	le       chan models.LE
)

func DB() *storm.DB {
	return db
}

func ConfigureServer(r chan bool, l chan models.LE) (*models.Settings, *negroni.Negroni, *storm.DB) {
	var err error
	reboot = r
	le = l
	gob.Register(&state.App{})
	box = packr.New(state.AssetsFolder, "../static")

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
	settings, err = models.InitSettings(helpers.DataDir())
	if err != nil {
		log.Fatalf("Failed to init settings : %s", err)
	}

	fss = sessions.NewFilesystemStore(state.SessionLocation(), settings.SessionHash, settings.SessionBlock)

	rtr := mux.NewRouter()
	rtr.HandleFunc("/assets/{theme}/{dir}/{file}", Asset).Methods("GET")
	rtr.HandleFunc("/setup", GetSetup).Methods("GET")
	rtr.HandleFunc("/setup", PostSetup).Methods("POST")
	rtr.HandleFunc("/logout", PostLogout).Methods("POST")
	rtr.HandleFunc("/t/{id}/i.png", GetTokenEvent).Methods("GET")
	rtr.HandleFunc("/", IndexCheckr).Methods("GET")
	rtr.NotFoundHandler = &NotFound{}

	// login has its own middleware chain
	login := mux.NewRouter().PathPrefix("/login").Subrouter().StrictSlash(false)
	login.Use(forceSetup)
	login.HandleFunc("", GetLogin).Methods("GET")
	login.HandleFunc("", PostLogin).Methods("POST")

	api := mux.NewRouter().PathPrefix("/api/v1").Subrouter().StrictSlash(false)
	api.Use(forceSetup)
	api.Use(isAuthenticatedWithToken)
	api.HandleFunc("/config.json", GetConfig).Methods("GET")
	api.HandleFunc("/event.json", PostEvent).Methods("POST")
	api.HandleFunc("/database/{dbname}", GetDatabase).Methods("GET")
	// protected, can't process unless logged in and setup is complete
	kushtaka := mux.NewRouter().PathPrefix("/kushtaka").Subrouter().StrictSlash(true)
	kushtaka.Use(forceSetup)
	kushtaka.Use(isAuthenticated)
	kushtaka.HandleFunc("/dashboard/page/{pid}/limit/{oid}", GetDashboard).Methods("GET")

	// clones
	kushtaka.HandleFunc("/clones/page/{pid}/limit/{oid}", GetClones).Methods("GET")
	kushtaka.HandleFunc("/clones", PostClones).Methods("POST")
	kushtaka.HandleFunc("/clone", DeleteClone).Methods("DELETE")

	// sensors
	kushtaka.HandleFunc("/sensors/page/{pid}/limit/{oid}", GetSensors).Methods("GET")
	kushtaka.HandleFunc("/sensors", PostSensors).Methods("POST")

	// sensor
	kushtaka.HandleFunc("/sensor/{id}", GetSensor).Methods("GET")
	kushtaka.HandleFunc("/sensor", PostSensor).Methods("POST")
	kushtaka.HandleFunc("/sensor", DeleteSensor).Methods("DELETE")

	// service
	kushtaka.HandleFunc("/service/{sensor_id}/type/{type}", PostService).Methods("POST")
	kushtaka.HandleFunc("/service", DeleteService).Methods("DELETE")
	kushtaka.HandleFunc("/service/team/update", UpdateSensorsTeam).Methods("PUT")

	// tokens
	kushtaka.HandleFunc("/tokens/page/{pid}/limit/{oid}", GetTokens).Methods("GET")
	kushtaka.HandleFunc("/tokens", PostTokens).Methods("POST")

	kushtaka.HandleFunc("/download/token/docx/{id}", DownloadDocxToken).Methods("GET")
	kushtaka.HandleFunc("/download/token/pdf/{id}", DownloadPdfToken).Methods("GET")
	// token
	kushtaka.HandleFunc("/token/{id}", GetToken).Methods("GET")
	kushtaka.HandleFunc("/token", PostToken).Methods("POST")
	kushtaka.HandleFunc("/token", PutToken).Methods("PUT")
	kushtaka.HandleFunc("/token", DeleteToken).Methods("DELETE")

	// smtp
	kushtaka.HandleFunc("/smtp", GetSmtp).Methods("GET")
	kushtaka.HandleFunc("/smtp", PostSmtp).Methods("POST")
	kushtaka.HandleFunc("/smtp/test", PostSendTestEmail).Methods("POST")

	// users
	kushtaka.HandleFunc("/users/page/{pid}/limit/{oid}", GetUsers).Methods("GET")
	kushtaka.HandleFunc("/users", PostUsers).Methods("POST")

	// user
	kushtaka.HandleFunc("/user/{id}", GetUser).Methods("GET")
	kushtaka.HandleFunc("/user/{id}", PostUser).Methods("POST")
	kushtaka.HandleFunc("/user/{id}", PutUser).Methods("PUT")
	kushtaka.HandleFunc("/user", DeleteUser).Methods("DELETE")

	// teams
	kushtaka.HandleFunc("/teams/page/{pid}/limit/{oid}", GetTeams).Methods("GET")
	kushtaka.HandleFunc("/teams", PostTeams).Methods("POST")

	// team
	kushtaka.HandleFunc("/team/{id}", GetTeam).Methods("GET")
	kushtaka.HandleFunc("/team/{id}", PostTeam).Methods("POST")
	kushtaka.HandleFunc("/team/{id}", PutTeam).Methods("PUT")
	kushtaka.HandleFunc("/team", DeleteTeam).Methods("DELETE")

	kushtaka.HandleFunc("/team/member/{id}", DeleteTeamMember).Methods("DELETE")

	// https
	kushtaka.HandleFunc("/https", GetHttps).Methods("GET")
	kushtaka.HandleFunc("/https/test", PostTestFQDN).Methods("POST")
	kushtaka.HandleFunc("/https/reboot", PostIRebootFQDN).Methods("POST")

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

	rtr.HandleFunc("/ws", Ws)

	// setup router
	n := negroni.New()
	n.Use(negroni.HandlerFunc(logHTTP))
	n.Use(negroni.HandlerFunc(Before))
	n.UseHandler(rtr)
	n.Use(negroni.HandlerFunc(After))

	return settings, n, db
}

// forceSetup is a middleware function that makes sure
// a admin user is created before allowing the user to
// proceed with using the application
func forceSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app, err := state.Restore(r)
		if err != nil {
			app.Fail("You must create an admin user before proceeding.")
			http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
		}

		var user models.User
		err = app.DB.One("ID", 1, &user)
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
		app, err := state.Restore(r)
		if err != nil {
			app.Fail("You must login before proceeding.")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		}

		if app.User.ID < 1 {
			app.Fail("You must login before proceeding.")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func logHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	host, port, _ := net.SplitHostPort(r.RemoteAddr)
	log.Infof("Duration: %s, Addr: %s, AddrPort: %s, Hostname: %s, Method: %s, Path: %s", time.Since(start), host, port, r.Host, r.Method, r.URL.Path)
	next.ServeHTTP(w, r)
}

func isAuthenticatedWithToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey string
		app, err := state.Restore(r)
		if err != nil {
			log.Fatal(err)
		}
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

func Before(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	// setup session and if it errors, create a new session
	sess, err := fss.Get(r, state.SessionName)
	if err != nil {
		fss.New(r, state.SessionName)
		sess, err = fss.Get(r, state.SessionName)
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

	ctx := context.WithValue(r.Context(), state.AppStateKey, app)
	next(w, r.WithContext(ctx))
}

func After(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	app := r.Context().Value(state.AppStateKey).(*state.App)

	// because we build the view upon each request
	// we clear it here to keep consistency and state

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
