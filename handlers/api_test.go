package handlers

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/models"
)

func createMockSensor(db *storm.DB) (*models.Sensor, error) {
	sensor := models.NewSensor("test_name", "test_note", 1)
	err := db.Save(&sensor)
	if err != nil {
		return nil, err
	}

	return sensor, nil
}

func TestGetMissingAPIKey(t *testing.T) {
	query := "/api/v1/sensor.json"
	srv, client, db := Buildup(t)
	defer srv.Close()
	defer db.Close()

	/*
		req := &http.Request{


		}
		client.D
	*/

	resp, err := client.Get(srv.URL + query)
	if err != nil {
		t.Fatal(err)
	}

	if status := resp.StatusCode; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(resp.Body)


	// TODO: mock the sensor, get the apikey, perform request with the apikey

	/*
		sensor, err := createMockSensor(db)
		if err != nil {
			t.Error(err)
		}

	targets := []string{
		"test@example.com",
	}

	for _, target := range targets {
		if !strings.Contains(buf.String(), target) {
			t.Errorf("The target [%s] was not found > %s", target, buf.String())
		}
	}
	*/

	Teardown()
}
