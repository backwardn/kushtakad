package service

import (
	"testing"
	"time"

	"github.com/kushtaka/kushtakad/models"
)

func TestNilSensor(t *testing.T) {

	var angel *ServiceAngel
	var sensor *models.Sensor

	v := shouldCancel(angel, sensor)
	if v != false {
		t.Errorf("Expected false, got %t", v)
	}
	sensor = &models.Sensor{}
	angel = &ServiceAngel{}
	angel.Sensor = &models.Sensor{}

	sensor.Updated = time.Now()
	angel.Sensor.Updated = time.Now()
	v = shouldCancel(angel, sensor)
	if v != false {
		t.Errorf("Expected false, got %t", v)
	}

	angel.Sensor.Updated = time.Now()
	sensor.Updated = time.Now()
	v = shouldCancel(angel, sensor)
	if v != true {
		t.Errorf("Expected true, got %t", v)
	}

}
