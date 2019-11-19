package handlers

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetLogin(t *testing.T) {
	srv, client, db := NewTestApp(t)
	defer srv.Close()
	defer db.Close()

	resp, err := client.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)

	targets := []string{
		"Password",
		"Email",
	}

	for _, target := range targets {
		if !strings.Contains(buf.String(), target) {
			t.Errorf("The target [%s] was not found > %s", target, buf.String())
		}
	}

	Teardown()
}

func TestPostLoginPasswordTooShort(t *testing.T) {
	srv, client, db := NewTestApp(t)
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

	buf.ReadFrom(resp.Body)
	target := "Password: must be between 12-64 characters."
	if !strings.Contains(buf.String(), target) {
		t.Errorf("The target [%s] was not found > %s", target, buf.String())
	}

	Teardown()
}

func TestPostLoginPasswordIncorrect(t *testing.T) {
	srv, client, db := NewTestApp(t)
	defer srv.Close()
	defer db.Close()

	buf := &bytes.Buffer{}
	v := url.Values{}
	v.Set("email", "test@example.com")
	v.Set("password", "123456789123")
	resp, err := client.PostForm(srv.URL+"/login", v)
	if err != nil {
		t.Error(err)
	}

	buf.ReadFrom(resp.Body)
	target := "User or password is incorrect."
	if !strings.Contains(buf.String(), target) {
		t.Errorf("The target [%s] was not found > %s", target, buf.String())
	}

	Teardown()
}
