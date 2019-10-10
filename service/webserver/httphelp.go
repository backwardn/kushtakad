package webserver

import "net/http"

type Res struct {
	ID         int64  `storm:"id,increment"`
	StatusCode int    `storm:"index"`
	URL        string `storm:"index,unique"`
	Headers    http.Header
	Body       []byte
	Orig       []byte
}

type Redirect struct {
	ID         int64  `storm:"id,increment"`
	StatusCode int    `storm:"index"`
	URL        string `storm:"index,unique"`
	GotoURL    string `storm:"index"`
	Headers    http.Header
}

/*
import (
	"net/http"
	"strings"

	"github.com/asdine/storm"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)
var db *storm.DB

func main() {
	var err error

	db, err = storm.Open("./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rtr := mux.NewRouter()
	rtr.PathPrefix("/").HandlerFunc(defaultHandler)
	n := negroni.New()
	n.UseHandler(rtr)
	log.Fatal(http.ListenAndServe("localhost:3002", n))

}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	var redir Redirect
	var res Res

	u := r.URL.RequestURI()

	db.One("URL", u, &redir)

	if redir.ID > 0 {
		for k, v := range redir.Headers {
			var s string
			for _, v1 := range v {
				v1 = strings.ReplaceAll(v1, "KUSHTAKA_URL_REPLACE", "localhost:3002")
				v1 = strings.ReplaceAll(v1, "https", "http")
				s = s + v1
			}
			w.Header().Set(k, s)
		}
		w.WriteHeader(redir.StatusCode)
		return
	}

	db.One("URL", u, &res)

	for k, v := range res.Headers {
		var s string
		for _, v1 := range v {
			v1 = strings.ReplaceAll(v1, "KUSHTAKA_URL_REPLACE", "localhost:3002")
			v1 = strings.ReplaceAll(v1, "https", "http")
			s = s + v1
		}

		switch strings.TrimSpace(k) {
		case "Strict-Transport-Security":
			//log.Printf("REMOVED Key: %s, Value: %s", k, s)
		case "Content-Length":
			//log.Printf("REMOVED Key: %s, Value: %s", k, s)
			//w.Header().Set(k, string(len(res.Body)))
		default:
			//log.Printf("ADDED Key: %s, Value: %s", k, s)
			w.Header().Set(k, s)
		}
	}

	if len(res.Body) == 0 {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
	w.Write(replaceURL(res.Body))
	return
}

func replaceURL(body []byte) []byte {
	s := string(body)

	// first replace this string
	s = strings.ReplaceAll(s, "https://KUSHTAKA_URL_REPLACE", "KUSHTAKA_URL_REPLACE")
	// then replace this
	s = strings.ReplaceAll(s, "http://KUSHTAKA_URL_REPLACE", "KUSHTAKA_URL_REPLACE")
	// now having normalized the links, replace them all with localhost
	s = strings.ReplaceAll(s, "KUSHTAKA_URL_REPLACE", "http://localhost:3002")
	return []byte(s)
}
*/
