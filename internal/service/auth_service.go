package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/allifiz/go-opname-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrUnauthorized = errors.New("unauthorized")

type AuthService struct {
	store  *repository.Store
	secret []byte
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthClaims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

func NewAuthService(store *repository.Store, secret string) *AuthService {
	return &AuthService{store: store, secret: []byte(secret)}
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (map[string]any, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" || input.Password == "" {
		return nil, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}
	user, err := s.store.GetUserByEmailWithRole(ctx, email)
	if err != nil || !user.IsActive {
		return nil, ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return nil, ErrUnauthorized
	}

	expiresAt := time.Now().UTC().Add(8 * time.Hour)
	claims := AuthClaims{UserID: user.ID.String(), Role: user.RoleCode, Email: user.Email, Exp: expiresAt.Unix()}
	token, err := SignAuthToken(claims, s.secret)
	if err != nil { return nil, err }
	return map[string]any{
		"access_token": token,
		"token_type": "Bearer",
		"expires_at": expiresAt,
		"user": map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.RoleCode},
	}, nil
}

func SignAuthToken(claims AuthClaims, secret []byte) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg":"HS256","typ":"JWT"})
	payload, err := json.Marshal(claims); if err != nil { return "", err }
	enc := base64.RawURLEncoding
	h := enc.EncodeToString(header)
	p := enc.EncodeToString(payload)
	unsigned := h + "." + p
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil)), nil
}

func ParseAuthToken(token string, secret []byte) (AuthClaims, error) {
	var claims AuthClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return claims, ErrUnauthorized }
	unsigned := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2]); if err != nil { return claims, ErrUnauthorized }
	mac := hmac.New(sha256.New, secret); _, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(sig, mac.Sum(nil)) { return claims, ErrUnauthorized }
	payload, err := base64.RawURLEncoding.DecodeString(parts[1]); if err != nil { return claims, ErrUnauthorized }
	if json.Unmarshal(payload, &claims) != nil || claims.UserID == "" || claims.Role == "" || claims.Exp <= time.Now().UTC().Unix() { return AuthClaims{}, ErrUnauthorized }
	return claims, nil
}
