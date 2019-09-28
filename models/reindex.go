package models

import (
	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/events"
)

// On startup reindex is called, though a bit dangerous, this is required
// should we change our datastructures, we want that change reflected in our data storage
func Reindex(db *storm.DB) error {
	var err error
	// User
	err = db.Init(&User{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&User{})
	if err != nil {
		return err
	}

	// EventManager
	err = db.Init(&events.EventManager{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&events.EventManager{})
	if err != nil {
		return err
	}

	// Smtp
	err = db.Init(&Smtp{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&Smtp{})
	if err != nil {
		return err
	}

	// Sensor
	err = db.Init(&Sensor{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&Sensor{})
	if err != nil {
		return err
	}

	// Token
	err = db.Init(&Token{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&Token{})
	if err != nil {
		return err
	}

	// Team
	err = db.Init(&Team{})
	if err != nil {
		return err
	}

	err = db.ReIndex(&Team{})
	if err != nil {
		return err
	}

	return nil
}
