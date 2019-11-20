package handlers

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
)

func Teardown() {
	os.RemoveAll(helpers.TestDataDir)
}

func Buildup(t *testing.T) (*httptest.Server, *http.Client, *storm.DB) {

	os.Setenv("KUSHTAKA_ENV", "test")

	Teardown()

	reboot := make(chan bool)
	le := make(chan models.LE)
	_, n, db := ConfigureServer(reboot, le)
	defer close(reboot)
	defer close(le)

	srv := httptest.NewServer(n)

	user := &models.User{
		Email:           "test@example.com",
		Password:        "testpassword1234",
		PasswordConfirm: "testpassword1234",
	}
	user.CreateAdmin(db)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Error(err)
	}
	client := srv.Client()
	client.Jar = jar

	return srv, client, db
}
