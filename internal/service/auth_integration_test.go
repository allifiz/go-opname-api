package service

import (
	"context"
	"testing"
	"time"

	"github.com/allifiz/go-opname-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthLoginIssuesValidToken(t *testing.T) {
	pool := integrationPool(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret-test-password"), bcrypt.DefaultCost)
	if err != nil { t.Fatal(err) }
	email := uniqueName("auth") + "@test.local"
	userID := scalarID(t, pool, `
		INSERT INTO users(role_id,name,email,password_hash)
		SELECT id,$1,$2,$3 FROM roles WHERE code='AKUNTAN'
		RETURNING id::text`, uniqueName("auth-user"), email, string(hash))

	secret := "01234567890123456789012345678901-test-secret"
	svc := NewAuthService(repository.NewStore(pool), secret)
	result, err := svc.Login(context.Background(), LoginInput{Email: email, Password: "secret-test-password"})
	if err != nil { t.Fatal(err) }
	token, ok := result["access_token"].(string)
	if !ok || token == "" { t.Fatal("access token missing") }
	claims, err := ParseAuthToken(token, []byte(secret))
	if err != nil { t.Fatal(err) }
	if claims.UserID != userID || claims.Role != "AKUNTAN" || claims.Email != email { t.Fatalf("unexpected claims: %+v", claims) }
}

func TestAuthTokenRejectsTamperingAndExpiry(t *testing.T) {
	secret := []byte("01234567890123456789012345678901-test-secret")
	token, err := SignAuthToken(AuthClaims{UserID:"user", Role:"CHEF", Email:"chef@test.local", Exp:time.Now().Add(time.Hour).Unix()}, secret)
	if err != nil { t.Fatal(err) }
	if _, err := ParseAuthToken(token+"x", secret); err == nil { t.Fatal("tampered token must fail") }

	expired, err := SignAuthToken(AuthClaims{UserID:"user", Role:"CHEF", Email:"chef@test.local", Exp:time.Now().Add(-time.Minute).Unix()}, secret)
	if err != nil { t.Fatal(err) }
	if _, err := ParseAuthToken(expired, secret); err == nil { t.Fatal("expired token must fail") }
}
