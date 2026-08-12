package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/allifiz/go-opname-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func TestConcurrentBootstrapCreatesExactlyOneInitialUser(t *testing.T) {
	pool := integrationPool(t)
	execSQL(t, pool, `TRUNCATE TABLE users CASCADE`)

	const bootstrapToken = "01234567890123456789012345678901-bootstrap"
	svc := NewAuthService(
		repository.NewStore(pool),
		"01234567890123456789012345678901-test-secret",
		bootstrapToken,
	)

	inputs := []BootstrapInitialUserInput{
		{Name: "Initial Accountant", Email: "initial-accountant@test.local", Password: "strong-test-password", Role: "AKUNTAN"},
		{Name: "Initial Chef", Email: "initial-chef@test.local", Password: "strong-test-password", Role: "CHEF"},
	}

	start := make(chan struct{})
	errCh := make(chan error, len(inputs))
	var wg sync.WaitGroup
	for _, input := range inputs {
		input := input
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.BootstrapInitialUser(context.Background(), bootstrapToken, input)
			errCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	successes := 0
	closed := 0
	for err := range errCh {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrBootstrapClosed):
			closed++
		default:
			t.Fatalf("unexpected bootstrap error: %v", err)
		}
	}
	if successes != 1 || closed != 1 {
		t.Fatalf("expected one success and one closed bootstrap, got success=%d closed=%d", successes, closed)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one user, got %d", count)
	}

	var passwordHash string
	if err := pool.QueryRow(context.Background(), `SELECT password_hash FROM users LIMIT 1`).Scan(&passwordHash); err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("strong-test-password")) != nil {
		t.Fatal("bootstrap password was not stored as a matching bcrypt hash")
	}
}

func TestAuthLoginIssuesValidToken(t *testing.T) {
	pool := integrationPool(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret-test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	email := uniqueName("auth") + "@test.local"
	userID := scalarID(t, pool, `
		INSERT INTO users(role_id,name,email,password_hash)
		SELECT id,$1,$2,$3 FROM roles WHERE code='AKUNTAN'
		RETURNING id::text`, uniqueName("auth-user"), email, string(hash))

	secret := "01234567890123456789012345678901-test-secret"
	svc := NewAuthService(repository.NewStore(pool), secret, "")
	result, err := svc.Login(context.Background(), LoginInput{Email: email, Password: "secret-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		t.Fatal("access token missing")
	}
	claims, err := ParseAuthToken(token, []byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != userID || claims.Role != "AKUNTAN" || claims.Email != email {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAuthTokenRejectsTamperingAndExpiry(t *testing.T) {
	secret := []byte("01234567890123456789012345678901-test-secret")
	token, err := SignAuthToken(AuthClaims{UserID: "user", Role: "CHEF", Email: "chef@test.local", Exp: time.Now().Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAuthToken(token+"x", secret); err == nil {
		t.Fatal("tampered token must fail")
	}

	expired, err := SignAuthToken(AuthClaims{UserID: "user", Role: "CHEF", Email: "chef@test.local", Exp: time.Now().Add(-time.Minute).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAuthToken(expired, secret); err == nil {
		t.Fatal("expired token must fail")
	}
}
