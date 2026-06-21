package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domainAuth "gosample/internal/domain/auth"
	"gosample/internal/infrastructure/config"
)

const jwtExpiry = 24 * time.Hour

type jwtCustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secret []byte
}

func NewJWTService(cfg *config.Config) domainAuth.IJWTService {
	return &jwtService{secret: []byte(cfg.JWTSecret)}
}

func (s *jwtService) Sign(_ context.Context, userID, role string) (string, error) {
	claims := jwtCustomClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *jwtService) Verify(_ context.Context, tokenStr string) (*domainAuth.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtCustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domainAuth.ErrUnauthorized
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, domainAuth.ErrUnauthorized
	}

	claims, ok := token.Claims.(*jwtCustomClaims)
	if !ok {
		return nil, domainAuth.ErrUnauthorized
	}

	return &domainAuth.JWTClaims{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}
