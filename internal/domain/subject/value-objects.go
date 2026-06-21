package subject

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidID = errors.New("invalid subject ID")
	ErrEmptyName = errors.New("subject name cannot be empty")
)

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
