package models

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
)

type Token struct {
	ID     int64  `storm:"id,increment,index"`
	TeamID int64  `storm:"unique" json:"team_id"`
	Key    string `storm:"unique" json:"key"`
	Name   string `storm:"index,unique" json:"name"`
	Note   string `storm:"index" json:"note"`
	Type   string `storm:"index" json:"type"` // Weblink, Pdf, Docx

	TokenContext interface{}
}

func NewToken() *Token {
	return &Token{}
}

func (t *Token) Wash() {
	t.Name = strings.TrimSpace(t.Name)
	t.Name = Strip(t.Name)
}

func (t *Token) ValidateCreate() error {
	t.Wash()
	return validation.Errors{
		"Name": validation.Validate(
			&t.Name,
			validation.Required,
			validation.Length(4, 64).Error("must be between 4-64 characters")),
		"Note": validation.Validate(
			&t.Name,
			validation.Required,
			validation.Length(1, 3000).Error("must be between 1-3000 characters")),
		"Type": validation.Validate(
			&t.Name,
			validation.Required),
	}.Filter()
}
