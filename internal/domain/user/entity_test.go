package user_test

import (
	"testing"

	"gosample/internal/domain/user"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"invalid format", "invalid-email", true},
		{"empty email", "", true},
		{"spaces around email", "  test@example.com  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewName(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{"valid name", "John Doe", false},
		{"empty name", "", true},
		{"spaces only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewName(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserCreationAndUpdates(t *testing.T) {
	e, _ := user.NewEmail("john@example.com")
	n, _ := user.NewName("John")
	u := user.NewUser(e, n)

	if u.Email().String() != "john@example.com" {
		t.Errorf("expected email john@example.com, got %s", u.Email().String())
	}
	if u.Name().String() != "John" {
		t.Errorf("expected name John, got %s", u.Name().String())
	}

	// Update name
	n2, _ := user.NewName("Johnathan")
	u.UpdateName(n2)

	if u.Name().String() != "Johnathan" {
		t.Errorf("expected updated name Johnathan, got %s", u.Name().String())
	}
}
