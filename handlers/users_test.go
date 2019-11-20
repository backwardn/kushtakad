package handlers

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestGetUsersNoAuth makes sure that the middleware is functioning
// and that if you haven't logged in, it redirects you.
func TestGetUsersNoAuth(t *testing.T) {
	query := "/kushtaka/users/page/1/limit/100"
	srv, client, db := Buildup(t)
	defer srv.Close()
	defer db.Close()

	resp, err := client.Get(srv.URL + query)
	if err != nil {
		t.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)

	targets := []string{
		"Email",
		"Password",
		"You must login before proceeding.",
	}

	for _, target := range targets {
		if !strings.Contains(buf.String(), target) {
			t.Errorf("The target [%s] was not found > %s", target, buf.String())
		}
	}

	Teardown()
}

func TestGetUsersEqualsOne(t *testing.T) {
	query := "/kushtaka/users/page/1/limit/100"
	srv, client, db := Buildup(t)
	defer srv.Close()
	defer db.Close()

	buf := &bytes.Buffer{}

	v := url.Values{}
	v.Set("email", "test@example.com")
	v.Set("password", "test")
	resp, err := client.PostForm(srv.URL+"/login", v)
	if err != nil {
		t.Error(err)
	}

	buf.Reset()

	resp, err = client.Get(srv.URL + query)
	if err != nil {
		t.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	buf.ReadFrom(resp.Body)

	targets := []string{
		"test@example.com",
	}

	for _, target := range targets {
		if !strings.Contains(buf.String(), target) {
			t.Errorf("The target [%s] was not found > %s", target, buf.String())
		}
	}

	Teardown()
}
