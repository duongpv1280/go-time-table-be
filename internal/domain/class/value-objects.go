package class

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidClassID = errors.New("invalid class ID")
	ErrEmptyClassName = errors.New("class name cannot be empty")
	ErrInvalidGrade   = errors.New("grade must be positive")
	ErrClassNotFound  = errors.New("class not found")
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
		return ID{}, ErrInvalidClassID
	}
	return ID{value: val}, nil
}

func (id ID) String() string {
	return id.value.String()
}

func (id ID) UUID() uuid.UUID {
	return id.value
}

type Name struct {
	value string
}

func NewName(s string) (Name, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return Name{}, ErrEmptyClassName
	}
	return Name{value: trimmed}, nil
}

func (n Name) String() string {
	return n.value
}

type Grade struct {
	value int
}

func NewGrade(g int) (Grade, error) {
	if g <= 0 {
		return Grade{}, ErrInvalidGrade
	}
	return Grade{value: g}, nil
}

func (g Grade) Value() int {
	return g.value
}
