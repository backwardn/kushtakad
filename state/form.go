package state

import "github.com/kushtaka/kushtakad/models"

type Forms struct {
	Setup      *models.User
	Login      *models.User
	Smtp       *models.Smtp
	Team       *models.Team
	TeamMember *models.Team
	Token      *models.Token
	Sensor *models.Sensor
}

func NewForms() *Forms {
	return &Forms{
		Setup:      &models.User{},
		Login:      &models.User{},
		Smtp:       &models.Smtp{},
		Team:       &models.Team{},
		TeamMember: &models.Team{},
		Token:      &models.Token{},
		Sensor:      &models.Sensor{},
	}
}
