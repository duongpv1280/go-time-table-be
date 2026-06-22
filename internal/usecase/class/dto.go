package class

import "time"

type ClassDTO struct {
	ID        string
	Name      string
	Grade     int
	CreatedAt time.Time
	UpdatedAt time.Time
}
