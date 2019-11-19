package handlers

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
)

func TestMain(m *testing.M) {
	os.Setenv("KUSHTAKA_ENV", "test")
	os.Exit(m.Run())
}

func Teardown(t *testing.T) {
	err := os.RemoveAll(helpers.TestDataDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostLogin(t *testing.T) {

	reboot := make(chan bool)
	le := make(chan models.LE)
	_, n, _ := ConfigureServer(reboot, le)
	defer close(reboot)
	defer close(le)

	data := url.Values{}
	data.Set("Email", "foo")
	data.Add("Password", "bar")
	b := bytes.NewBufferString(data.Encode())

	req := httptest.NewRequest("POST", "/login", b)
	rr := httptest.NewRecorder()

	n.ServeHTTP(rr, req)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	sbody := string(body)
	log.Debug(sbody)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	/*
		res, err := http.Get("http://localhost/login")
		if err != nil {
			t.Fatal(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		sbody := string(body)

		want := []string{
			"incorrect",
		}

		for _, target := range want {
			if !strings.Contains(sbody, target) {
				t.Errorf("Target '%s' not found in the response", target)
				t.Errorf("Body is \n %s", sbody)
			}
		}
	*/

	//Teardown(t)

}

func TestGetLogin(t *testing.T) {
	reboot := make(chan bool)
	le := make(chan models.LE)
	_, n, _ := ConfigureServer(reboot, le)
	defer close(reboot)
	defer close(le)

	user := &models.User{
		Email:           "test@example.com",
		Password:        "testpassword1234",
		PasswordConfirm: "testpassword1234",
	}
	err := user.CreateAdmin(DB())
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	n.ServeHTTP(rr, r)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Fatal(err)
	}
	sbody := string(body)

	want := []string{
		"Password",
		"Email",
	}

	for _, target := range want {
		if !strings.Contains(sbody, target) {
			t.Errorf("Target '%s' not found in the response", target)
			t.Errorf("Body is \n %s", sbody)
		}
	}

	err = os.RemoveAll(helpers.TestDataDir)
	if err != nil {
		t.Fatal(err)
	}
}
