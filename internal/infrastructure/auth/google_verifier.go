package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domainAuth "gosample/internal/domain/auth"
)

type GoogleVerifier struct {
	client *http.Client
}

func NewGoogleVerifier() domainAuth.IGoogleVerifier {
	return &GoogleVerifier{client: &http.Client{}}
}

func (v *GoogleVerifier) Verify(ctx context.Context, idToken string) (*domainAuth.GoogleClaims, error) {
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified string `json:"email_verified"`
		Error         string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || body.Error != "" || body.Email == "" {
		return nil, domainAuth.ErrInvalidToken
	}

	return &domainAuth.GoogleClaims{
		Sub:   body.Sub,
		Email: body.Email,
		Name:  body.Name,
	}, nil
}
