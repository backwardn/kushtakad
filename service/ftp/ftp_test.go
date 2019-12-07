// Copyright 2016-2019 DutchSec (https://dutchsec.com/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ftp

import (
	"net"
	"os"
	"testing"

	"github.com/asdine/storm"
	"github.com/gobuffalo/packr/v2"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/state"
)

const (
	user     = "anonymous"
	password = "anonymous"
)

var (
	clt, srv net.Conn
)

func Teardown() {
	os.RemoveAll(helpers.TestDataDir)
}

func Buildup(t *testing.T) *storm.DB {

	os.Setenv("KUSHTAKA_ENV", "test")

	Teardown()

	box := packr.New("static", "../static")

	err := state.SetupFileStructure(box)
	if err != nil {
		t.Error(err)
	}

	db, err := storm.Open(state.DbLocation())
	if err != nil {
		t.Error(err)
	}

	err = models.Reindex(db)
	if err != nil {
		t.Error(err)
	}

	// must setup the basic hashes and settings for application to function
	_, err = models.InitSettings(helpers.DataDir())
	if err != nil {
		t.Error(err)
	}

	return db
}

func TestFTP(t *testing.T) {

	//Setup client and server
	db := Buildup(t)
	clt, srv = net.Pipe()
	defer clt.Close()
	defer srv.Close()

	s := &FtpService{}
	s.ConfigureAndRun()

	//Handle the connection
	go func(conn net.Conn) {
		if err := s.Handle(nil, conn, db); err != nil {
			t.Fatal(err)
		}
	}(srv)

	client, err := Connect(clt)
	if err != nil {
		t.Fatal(err)
	}
	//log.Debug("Test: client started")

	if err := client.Login(user, password); err != nil {
		t.Errorf("Could not login user: %s password: %s", user, password)
	}

	if err := client.Quit(); err != nil {
		t.Errorf("Error with Quit: %s", err.Error())
	}
}
