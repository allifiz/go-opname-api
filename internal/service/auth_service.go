package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/allifiz/go-opname-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrBootstrapDisabled     = errors.New("bootstrap disabled")
	ErrBootstrapUnauthorized = errors.New("bootstrap unauthorized")
	ErrBootstrapClosed       = errors.New("bootstrap closed")
)

type AuthService struct {
	store          *repository.Store
	secret         []byte
	bootstrapToken string
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type BootstrapInitialUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type AuthClaims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

func NewAuthService(store *repository.Store, secret, bootstrapToken string) *AuthService {
	return &AuthService{store: store, secret: []byte(secret), bootstrapToken: bootstrapToken}
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
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expiresAt,
		"user":         map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.RoleCode},
	}, nil
}

func (s *AuthService) BootstrapInitialUser(ctx context.Context, token string, input BootstrapInitialUserInput) (map[string]any, error) {
	if s.bootstrapToken == "" {
		return nil, ErrBootstrapDisabled
	}
	if token == "" || !secureStringEqual(token, s.bootstrapToken) {
		return nil, ErrBootstrapUnauthorized
	}

	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(strings.ToLower(input.Email))
	role := strings.ToUpper(strings.TrimSpace(input.Role))
	if name == "" || len(name) > 150 {
		return nil, fmt.Errorf("%w: name is required and must be at most 150 characters", ErrInvalidInput)
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 150 {
		return nil, fmt.Errorf("%w: valid email is required", ErrInvalidInput)
	}
	if len(input.Password) < 12 || len(input.Password) > 72 {
		return nil, fmt.Errorf("%w: password must contain 12 to 72 characters", ErrInvalidInput)
	}
	if !isBootstrapRole(role) {
		return nil, fmt.Errorf("%w: unsupported role", ErrInvalidInput)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := s.store.BootstrapInitialUser(ctx, name, email, string(passwordHash), role)
	if err != nil {
		if errors.Is(err, repository.ErrInitialUserAlreadyExists) {
			return nil, ErrBootstrapClosed
		}
		if errors.Is(err, repository.ErrRoleNotFound) {
			return nil, fmt.Errorf("%w: unsupported role", ErrInvalidInput)
		}
		return nil, err
	}

	return map[string]any{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"role":      role,
		"is_active": user.IsActive,
	}, nil
}

func secureStringEqual(a, b string) bool {
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}

func isBootstrapRole(role string) bool {
	switch role {
	case "CHEF", "AHLI_GIZI", "PENGAWAS_BAHAN_BAKU", "AKUNTAN", "KEPALA_SPPG":
		return true
	default:
		return false
	}
}

func SignAuthToken(claims AuthClaims, secret []byte) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
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
	if len(parts) != 3 {
		return claims, ErrUnauthorized
	}
	unsigned := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, ErrUnauthorized
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return claims, ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrUnauthorized
	}
	if json.Unmarshal(payload, &claims) != nil || claims.UserID == "" || claims.Role == "" || claims.Exp <= time.Now().UTC().Unix() {
		return AuthClaims{}, ErrUnauthorized
	}
	return claims, nil
}
