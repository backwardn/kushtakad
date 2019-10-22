package helpers

import (
	//stdlib

	"bytes"
	"fmt"
	"net/smtp"
	"path/filepath"
	"text/template"

	"github.com/asdine/storm"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/kushtaka/kushtakad/models"
	email "gopkg.in/jordan-wright/email.v1"
)

const smtpID = 1

type Mailer struct {
	Smtp     *models.Smtp
	Settings *models.Settings
	DB       *storm.DB
	Box      *packr.Box

	EventID      int64
	EventLink    string
	Subject      string
	Text         string
	TemplateName string
	TemplateFile string
}

type SensorEvent struct {
	ID int64
}

type TokenEvent struct {
}

type Event struct {
	Mailer *Mailer

	Email *Email
	Link  string
}

type Email struct {
	Subject      string
	Body         string
	To           []string
	From         string
	TemplateName string
	Filename     string
	URI          string
}

func NewEvent(db *storm.DB, box *packr.Box) *Event {
	return &Event{
		Mailer: NewMailer(db, box),
		Email:  &Email{},
	}
}

func NewMailer(db *storm.DB, box *packr.Box) *Mailer {
	smtp := &models.Smtp{}
	err := db.One("ID", 1, smtp)
	if err != nil {
		log.Debugf("Smtp object not found in database %v", err)
	}

	m := &Mailer{Smtp: smtp, Box: box, DB: db}
	return m
}

func buildTemplate(email *Email, box *packr.Box) ([]byte, error) {
	var out bytes.Buffer
	fullPath := filepath.Join("admin", "email", email.Filename)
	templateBytes, err := box.Find(fullPath)
	if err != nil {
		return nil, err
	}

	t, err := template.New("MT").Parse(string(templateBytes))
	if err != nil {
		return nil, err
	}

	err = t.ExecuteTemplate(&out, email.TemplateName, email)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func (te *Event) SendEvent() error {
	tmpl, err := buildTemplate(te.Email, te.Mailer.Box)
	if err != nil {
		return err
	}

	e := email.NewEmail()
	e.From = te.Mailer.Smtp.Email
	e.To = te.Email.To
	e.Subject = te.Email.Subject
	e.HTML = tmpl
	if te.Mailer.Smtp.Username != "" {
		err = e.Send(
			fmt.Sprintf("%s:%s", te.Mailer.Smtp.Host, te.Mailer.Smtp.Port),
			smtp.PlainAuth(
				"",
				te.Mailer.Smtp.Username,
				te.Mailer.Smtp.Password,
				te.Mailer.Smtp.Host,
			),
		)
	} else {
		err = e.Send(
			fmt.Sprintf("%s:%s", te.Mailer.Smtp.Host, te.Mailer.Smtp.Port),
			nil,
		)
	}
	log.Debug(err)
	return err

}
