package user

import (
	"time"
)

// User is the Aggregate Root representing a user in the system.
type User struct {
	id        ID
	email     Email
	name      Name
	role      Role
	googleID  string
	createdAt time.Time
	updatedAt time.Time
}

// NewUser creates a new User instance (for new users, no role assigned).
func NewUser(email Email, name Name) *User {
	now := time.Now().UTC()
	return &User{
		id:        NewID(),
		email:     email,
		name:      name,
		createdAt: now,
		updatedAt: now,
	}
}

// NewGoogleUser creates a new User authenticated via Google OAuth.
func NewGoogleUser(email Email, name Name, role Role, googleID string) *User {
	now := time.Now().UTC()
	return &User{
		id:        NewID(),
		email:     email,
		name:      name,
		role:      role,
		googleID:  googleID,
		createdAt: now,
		updatedAt: now,
	}
}

// RestoreUser restores an existing User instance from storage.
func RestoreUser(id ID, email Email, name Name, role Role, googleID string, createdAt, updatedAt time.Time) *User {
	return &User{
		id:        id,
		email:     email,
		name:      name,
		role:      role,
		googleID:  googleID,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// Getters
func (u *User) ID() ID {
	return u.id
}

func (u *User) Email() Email {
	return u.email
}

func (u *User) Name() Name {
	return u.name
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) GoogleID() string {
	return u.googleID
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// Business Methods

// UpdateName updates the user's name and updates the updatedAt timestamp.
func (u *User) UpdateName(newName Name) {
	u.name = newName
	u.updatedAt = time.Now().UTC()
}

// UpdateEmail updates the user's email and updates the updatedAt timestamp.
func (u *User) UpdateEmail(newEmail Email) {
	u.email = newEmail
	u.updatedAt = time.Now().UTC()
}
