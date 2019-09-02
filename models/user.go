package models

import (
	"errors"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
	is "github.com/go-ozzo/ozzo-validation/is"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              int64  `storm:"id,increment,index"`
	Email           string `storm:"index,unique" json:"email"`
	Password        string `json:"-"`
	PasswordConfirm string `json:"-"`
	IsDisabled      bool   `storm:"index" json:"is_disabled"`
	Hash            string
}

func NewUser() *User {
	return &User{}
}

func (u *User) Wash() {
	u.Email = strings.TrimSpace(u.Email)
	u.Email = Strip(u.Email)
}

func (u User) ValidateLogin() error {
	u.Wash()
	return validation.Errors{
		"Email": validation.Validate(
			&u.Email,
			validation.Required,
			validation.Length(5, 128).Error("must be between 5-128 characters"),
			is.Email.Error("must be an email address")),
		"Password": validation.Validate(
			&u.Password,
			validation.Required,
			validation.Length(12, 64).Error("must be between 12-64 characters")),
	}.Filter()
}

func (u User) ValidateSetup() error {
	u.Wash()
	s := u.PasswordConfirm
	return validation.Errors{
		"Email": validation.Validate(
			&u.Email,
			validation.Required,
			validation.Length(5, 128).Error("must be between 5-128 characters"),
			is.Email.Error("must be an email address")),
		"Password": validation.Validate(
			&u.Password,
			validation.Required,
			validation.Length(12, 64).Error("must be between 12-64 characters"),
			validation.In(s).Error("does not match 'Password Confirm'")),
	}.Filter()
}

func (u *User) HashPassword() error {
	hpwd, err := bcrypt.GenerateFromPassword([]byte(u.Password), 13)
	if err != nil {
		return err
	}
	u.Hash = string(hpwd) // set hashed password
	return nil
}

func (u *User) Authenticate(password string) error {
	if len(u.Email) == 0 {
		return errors.New("User is not populated")
	}

	if len(u.Hash) < 12 {
		return errors.New("Password is not populated")
	}

	// make sure the supplied password and
	err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password))
	if err != nil {
		return err
	}

	return nil
}
