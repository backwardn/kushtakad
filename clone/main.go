package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/asdine/storm"
	"github.com/gocolly/colly"
	"github.com/gorilla/css/scanner"
	"github.com/lukasbob/srcset"
)

const KushtakaUrlPlaceholder = "KUSHTAKA_URL_REPLACE"

var db *storm.DB
var fa *ForceAssets
var replaceDomain *ForceAssets

func needToDownload(s string) {
	fa.mu.Lock()
	fa.Assets[s] = s
	fa.mu.Unlock()
}

func needToReplace(s string) {
	replaceDomain.mu.Lock()
	replaceDomain.Assets[s] = s
	replaceDomain.mu.Unlock()
}

func forceReplace() {
	replaceDomain.mu.Lock()
	var all []Res
	db.All(&all)
	for _, res := range all {
		for _, uri := range replaceDomain.Assets {
			body := strings.ReplaceAll(string(res.Body), uri, "")
			res.Body = []byte(body)
			/*
				for _, headers := range res.Headers {
					for _, header := range headers {
						log.Println(header, strings.ReplaceAll(header, uri, ""))
					}
				}
			*/
		}
		err := db.Update(&res)
		if err != nil {
			log.Fatal("Unable to update forceReplace()", err)
		}
	}
	replaceDomain.mu.Unlock()
}

//var DOMAIN = "www.bend.k12.or.us"
//var DOMAIN = "synergy.beaverton.k12.or.us"
//var DOMAIN = "www.beaverton.k12.or.us"
//var DOMAIN = "vpn.umass.edu"
//var PRIMARYLINK = "/global-protect/login.esp"
//var DOMAIN = "www.seattleschools.org"
//var SCHEME = "https://"
//var DOMAIN = "www.opsecedu.com"

var URI, SCHEME, DOMAIN, PRIMARYLINK string
var DEPTH int

type ForceAssets struct {
	mu     *sync.Mutex
	Assets map[string]string
}

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

func main() {

	var err error
	flag.StringVar(&URI, "uri", "", "you must specify a uri (https://example.com/test) that you would like kushtaka to mimic")
	flag.IntVar(&DEPTH, "depth", 1, "the default depth that kushtaka tries to crawl is set to 1, increase it should you want more assets to be mimic'd")
	flag.Parse()

	if len(URI) == 0 {
		log.Fatal("Must specify URI to scrape")
	}

	uri, err := url.Parse(URI)
	if err != nil {
		log.Fatal("Unable to parse URI to scrape", err)
	}

	log.Println(uri.Scheme)

	if !strings.Contains(uri.Scheme, "https") {
		log.Fatal("URI doesn't have a scheme (http/https)")
	} else {
		SCHEME = uri.Scheme + "://"
	}

	if len(uri.Hostname()) < 4 {
	} else {
		DOMAIN = uri.Hostname()
	}

	if len(uri.RequestURI()) > 1 {
		PRIMARYLINK = uri.RequestURI()
	} else {
		PRIMARYLINK = "/"
	}

	fa = &ForceAssets{mu: &sync.Mutex{}, Assets: make(map[string]string)}
	replaceDomain = &ForceAssets{mu: &sync.Mutex{}, Assets: make(map[string]string)}
	db, err = storm.Open("test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Instantiate default collector
	c := colly.NewCollector(
		colly.MaxDepth(DEPTH),
		colly.AllowedDomains(DOMAIN),
		colly.Async(true),
	)
	c.UserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:67.0) Gecko/20100101 Firefox/67.0"

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 8,
		RandomDelay: 200 * time.Millisecond,
	})

	// On every a element which has href attribute call callback
	c.OnHTML("style", func(e *colly.HTMLElement) {
		newcss := cssReplaceUrl(e.Text, e.Request.URL)
		s := strings.ReplaceAll(string(e.Response.Body), e.Text, newcss)
		e.Response.Body = []byte(s)
	})

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		err := e.Request.Visit(link)
		if err != nil {
			return
		}
	})

	c.OnHTML("link[href]", func(e *colly.HTMLElement) {
		src := e.Attr("href")
		err := e.Request.Visit(src)
		if err != nil {
			return
		}
		newsrc, err := absUrl(src, e.Request.URL)
		if err != nil {
			log.Println("can't download link[href]...", err)
			return
		}
		needToDownload(newsrc.String())
	})

	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		parsesrc, err := url.Parse(src)
		if err != nil {
			log.Println("can't download script[src]...", err)
			return
		}
		newsrc, err := absUrl(parsesrc.String(), e.Request.URL)
		if err != nil {
			log.Println("can't download script[src]...", err)
			return
		}
		needToDownload(newsrc.String())
	})

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		e.Request.Visit(src)
		newsrc, err := absUrl(src, e.Request.URL)
		if err != nil {
			log.Println("can't download img[src]...", err)
			return
		}
		needToDownload(newsrc.String())
	})

	c.OnHTML("img[srcset]", func(e *colly.HTMLElement) {
		srcs := e.Attr("srcset")
		imset := srcset.Parse(srcs)
		for _, src := range imset {
			newsrc, err := absUrl(src.URL, e.Request.URL)
			if err != nil {
				log.Println("can't download img[src]...", err)
				return
			}
			needToDownload(newsrc.String())
		}
	})

	c.OnRequest(func(r *colly.Request) {
	})

	c.OnResponse(func(r *colly.Response) {
		var body []byte
		u := r.Request.URL.RequestURI()

		// make root document if empty
		if len(u) == 0 {
			u = "/"
		}

		headers := replaceHeader(*r.Headers)
		body = r.Body
		contentType := http.DetectContentType(body)
		log.Println("OnReponse() ", u, contentType, r.Request.Depth)

		if len(r.Body) < 15 {
			log.Println("Body is empty moving on...")
			return
		}

		if isCss(headers) {
			body = []byte(cssReplaceUrl(string(r.Body), r.Request.URL))
		} else if strings.ContainsAny(contentType, "text/hml") {
			body = replaceURL(r.Body)
		}

		if len(body) < 15 {
			log.Fatal(u, "Body is STILL empty?")
			return
		}

		res := Res{
			Headers:    headers,
			StatusCode: r.StatusCode,
			URL:        u,
			Body:       body,
			Orig:       r.Body,
		}
		search := Res{}
		tx, err := db.Begin(true)
		if err != nil {
			log.Fatal("Unable to Begin() Tx", err)
		}
		defer tx.Rollback()

		tx.One("URL", u, &search)
		err = tx.Save(&res)
		switch err {
		case storm.ErrAlreadyExists:
			res.ID = search.ID
			err := tx.Update(&res)
			if err != nil {
				log.Fatal("Unable to Update() Tx", err)
			}
		}

		tx.Commit()

	})

	c.RedirectHandler = RedirectHandler
	err = c.Visit(SCHEME + DOMAIN + PRIMARYLINK)
	if err != nil {
		log.Fatal(err)
	}

	c.Wait()
	downloadAssets()
	forceReplace()

}

func downloadAssets() {
	fa.mu.Lock()
	for _, v := range fa.Assets {
		resp, err := http.Get(v)
		if err != nil {
			log.Println(err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("unable to read body", err)
			}
			uri := resp.Request.URL.ResolveReference(resp.Request.URL)
			u := uri.RequestURI()
			headers := replaceHeader(resp.Header)

			res := Res{
				URL:        u,
				Body:       body,
				Headers:    headers,
				StatusCode: resp.StatusCode,
			}

			search := Res{}
			tx, err := db.Begin(true)
			if err != nil {
				log.Fatal("Unable to Begin() Tx", err)
			}
			defer tx.Rollback()

			tx.One("URL", u, &search)
			err = tx.Save(&res)
			switch err {
			case storm.ErrAlreadyExists:
				res.ID = search.ID
				err := tx.Update(&res)
				if err != nil {
					log.Fatal("Unable to Update() Tx", err)
				}
			}

			tx.Commit()

			h := resp.Request.URL.Hostname()
			sh := resp.Request.URL.Scheme
			uril := sh + "://" + h
			log.Println("downloading Asset: ", uril, v)
			needToReplace(uril)

		}
	}
	fa.mu.Unlock()
}

// SetRedirectHandler instructs the Collector to allow multiple downloads of the same URL
func RedirectHandler(req *http.Request, via []*http.Request) error {
	redirUrl := "/" + strings.Trim(req.Referer(), SCHEME+DOMAIN)
	log.Println("REDIRECT", redirUrl)

	res := Redirect{
		Headers:    replaceHeader(req.Response.Header),
		StatusCode: req.Response.StatusCode,
		URL:        redirUrl,
		GotoURL:    req.URL.RequestURI(),
	}

	tx, err := db.Begin(true)
	if err != nil {
		log.Fatal("Unable to Begin() Tx", err)
	}
	defer tx.Rollback()

	search := Redirect{}
	tx.One("URL", redirUrl, &search)
	err = tx.Save(&res)
	switch err {
	case storm.ErrAlreadyExists:
		res.ID = search.ID
		err := tx.Update(&res)
		if err != nil {
			log.Fatal("Unable to Update() Tx", err)
		}
	}
	tx.Commit()

	return nil
}

func replaceURL(body []byte) []byte {
	s := string(body)
	s = strings.ReplaceAll(s, DOMAIN, KushtakaUrlPlaceholder)
	return []byte(s)
}

func isa(header string, headers http.Header) bool {
	for _, v := range headers {
		for _, v1 := range v {
			if v1 == header {
				return true
			}
		}
	}
	return false
}

func isCss(headers http.Header) bool {
	return isa("text/css", headers)
}

func isHtml(headers http.Header) bool {
	return isa("text/html", headers)
}

func absUrl(uri string, parent *url.URL) (*url.URL, error) {

	// clean the background: url() syntax from it
	uri = cleanCssUrl(uri)

	// data: if the data is embedded in the URI skip it
	if strings.Contains(uri, "data:") {
		return nil, errors.New("Uri contains an embedded data: asset")
	}

	if strings.HasPrefix(uri, "//") {
		uri = parent.Scheme + ":" + uri
	}

	if strings.ContainsAny(uri, "//..") {
		uri = strings.ReplaceAll(uri, "//..", "/..")
	}

	purl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	newuri := parent.ResolveReference(purl)

	return newuri, nil

}

func cleanCssUrl(v string) string {
	// normalize the string
	v = strings.ReplaceAll(v, "url(", "")
	v = strings.ReplaceAll(v, ")", "")
	v = strings.ReplaceAll(v, "'", "")
	v = strings.ReplaceAll(v, "\"", "")
	return v
}

func createCssLookup(css string, surl *url.URL) map[string]string {
	lookup := make(map[string]string)
	scan := scanner.New(css)
	done := false
	for !done {
		tok := scan.Next()
		switch tok.Type {
		case scanner.TokenEOF:
			done = true
		case scanner.TokenURI:

			// copy the data from the token
			uri, err := absUrl(tok.Value, surl)
			if err != nil {
				log.Println(err)
				break
			}

			needToDownload(uri.String())
			log.Println(uri.String())
			lookup[tok.Value] = "url(" + uri.RequestURI() + ")"

		}
	}

	return lookup
}

/*
func defineRootFolder(url string) string {
	log.Println("defineRootFolder:", url)
	dir, file := filepath.Split(url)
	log.Println("defineRootFolder:", dir, " file: ", file)
	return dir
}
*/

func cssReplaceUrl(css string, url *url.URL) string {
	if len(css) < 1 {
		log.Fatal("why is empyt?")
	}

	m := createCssLookup(css, url)
	for orig, change := range m {
		log.Println("cssReplaceUrl()")
		log.Println("\turl", url)
		log.Println("\torig", orig)
		log.Println("\tchange", change)
		css = strings.ReplaceAll(css, orig, change)
	}

	if len(css) < 1 {
		log.Fatal("why is empyt now?")
	}

	return css
}

func replaceHeader(header http.Header) http.Header {
	for k, m := range header {
		var headerstring string
		for _, v1 := range m {
			headerstring = headerstring + strings.ReplaceAll(v1, DOMAIN, KushtakaUrlPlaceholder)
		}
		header.Set(k, headerstring)
	}
	return header
}
