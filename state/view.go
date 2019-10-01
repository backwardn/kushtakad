package state

import (
	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/models"
)

// View is built, rendered, and cleared every HTTP request
type View struct {
	FlashSuccess   []string
	FlashFail      []string
	URI            string
	Crumbs         []*Crumb
	User           *models.User
	State          *models.State
	Links          *Links
	Forms          *Forms
	LETests        []models.LETest
	Team           *models.Team
	Teams          []models.Team
	Token          *models.Token
	Tokens         []models.Token
	Sensor         *models.Sensor
	Sensors        []models.Sensor
	Users          []models.User
	SensorServices []models.ServiceCfg
	Events         []events.EventManager
}

type Crumb struct {
	Name string
	Link string
}

type Links struct {
	Setup bool
	Login bool

	Dashboard string
	Tokens    string
	Sensors   string
	Users     string
	Teams     string
	Https     string
	Smtp      string
}

func NewView() *View {
	var ff []string
	var fs []string
	var tm []models.Team
	var users []models.User
	var crumbs []*Crumb
	var events []events.EventManager
	var letests []models.LETest
	return &View{
		LETests:      letests,
		FlashFail:    ff,
		FlashSuccess: fs,
		Teams:        tm,
		Events:       events,
		Users:        users,
		Crumbs:       crumbs,
		Links:        &Links{},
		Forms:        NewForms(),
		Team:         models.NewTeam(),
		Token:        models.NewToken(),
	}
}

func (v *View) AddCrumb(name, link string) {
	c := &Crumb{Name: name, Link: link}
	v.Crumbs = append(v.Crumbs, c)
}
