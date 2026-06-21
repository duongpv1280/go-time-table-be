package user

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrEmptyName    = errors.New("name cannot be empty")
	ErrInvalidID    = errors.New("invalid user ID")
)

// ID represents the unique identifier of a User.
type ID struct {
	value uuid.UUID
}

func NewID() ID {
	return ID{value: uuid.New()}
}

func ParseID(s string) (ID, error) {
	val, err := uuid.Parse(s)
	if err != nil {
		return ID{}, ErrInvalidID
	}
	return ID{value: val}, nil
}

func (id ID) String() string {
	return id.value.String()
}

func (id ID) UUID() uuid.UUID {
	return id.value
}

// Email represents a validated email address.
type Email struct {
	value string
}

func NewEmail(address string) (Email, error) {
	trimmed := strings.TrimSpace(address)
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: trimmed}, nil
}

func (e Email) String() string {
	return e.value
}

// Name represents a user's name.
type Name struct {
	value string
}

func NewName(val string) (Name, error) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return Name{}, ErrEmptyName
	}
	return Name{value: trimmed}, nil
}

func (n Name) String() string {
	return n.value
}
