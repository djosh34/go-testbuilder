package examples

import (
	"errors"
)

// System under test.
type UserController struct {
	Mailer     MailService
	Repository UserRepository
}

func (c *UserController) Handle(userName string, payload string) (*User, error) {
	if payload == "" {
		return nil, errors.New("invalid payload")
	}

	user, err := c.Repository.GetUser(userName)
	if err != nil {
		return nil, err
	}

	if err := c.Mailer.SendMail(); err != nil {
		return nil, err
	}

	if err := c.Repository.StoreUser(user); err != nil {
		return nil, err
	}

	return &user, nil
}


// Interfaces
type MailService interface {
	SendMail() error
}

type User struct {
	Name string
}

type UserRepository interface {
	GetUser(string) (User, error)
	StoreUser(User) error
}
