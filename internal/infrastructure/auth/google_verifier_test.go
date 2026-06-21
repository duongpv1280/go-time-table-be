package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainAuth "gosample/internal/domain/auth"
	infraAuth "gosample/internal/infrastructure/auth"
)

// TestGoogleVerifier_InvalidToken verifies that the real Google tokeninfo
// endpoint rejects a known-invalid token.
func TestGoogleVerifier_InvalidToken(t *testing.T) {
	verifier := infraAuth.NewGoogleVerifier()

	_, err := verifier.Verify(context.Background(), "this-is-not-a-valid-google-id-token")

	require.Error(t, err)
	assert.ErrorIs(t, err, domainAuth.ErrInvalidToken)
}
