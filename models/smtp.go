package models

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
	is "github.com/go-ozzo/ozzo-validation/is"
)

type Smtp struct {
	ID       int64 `storm:"id,increment,index"`
	Sender   string
	Email    string
	Host     string
	Port     string
	Username string
	Password string
}

func NewSmtp() *Smtp {
	return &Smtp{}
}

func (s *Smtp) Wash() {
	s.Sender = strings.TrimSpace(s.Sender)
	s.Sender = Strip(s.Sender)

	s.Email = strings.TrimSpace(s.Email)
	s.Email = Strip(s.Email)

	s.Host = strings.TrimSpace(s.Host)
	s.Host = Strip(s.Host)

	s.Port = strings.TrimSpace(s.Port)
	s.Port = Strip(s.Port)

	s.Username = strings.TrimSpace(s.Username)
	s.Username = Strip(s.Username)

	s.Password = strings.TrimSpace(s.Password)
	s.Password = Strip(s.Password)
}

func (s Smtp) ValidateSmtp() error {
	s.Wash()
	return validation.Errors{
		"Sender": validation.Validate(
			&s.Sender,
			validation.Required,
			validation.Length(5, 128).Error("must be between 5-128 characters")),
		"Email": validation.Validate(
			&s.Email,
			validation.Required,
			validation.Length(5, 128).Error("must be between 5-128 characters"),
			is.Email.Error("must be an email address")),
		"Host": validation.Validate(
			&s.Host,
			validation.Required,
			validation.Length(5, 128).Error("must be between 5-128 characters"),
			is.Host.Error("must be a valid ipv4, ipv6, or dns address")),
		"Port": validation.Validate(
			&s.Port,
			validation.Required,
			validation.Length(1, 5).Error("must be between 1-5 characters"),
			is.Port.Error("must be a valid port")),
	}.Filter()
}
