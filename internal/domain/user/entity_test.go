package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserCreationAndUpdates(t *testing.T) {
	e, err := user.NewEmail("john@example.com")
	require.NoError(t, err)

	n, err := user.NewName("John")
	require.NoError(t, err)

	u := user.NewUser(e, n)

	assert.Equal(t, "john@example.com", u.Email().String())
	assert.Equal(t, "John", u.Name().String())

	n2, err := user.NewName("Johnathan")
	require.NoError(t, err)
	u.UpdateName(n2)

	assert.Equal(t, "Johnathan", u.Name().String())
}
