package auth

import (
	"testing"
	"time"

	"github.com/egorbr4z/panel/internal/models"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword("s3cret-pass", hash)
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}
	bad, _ := VerifyPassword("wrong-pass", hash)
	if bad {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestJWTIssueParse(t *testing.T) {
	m := NewManager("test-secret", 15*time.Minute, 24*time.Hour)
	admin := &models.Admin{ID: 7, Username: "root", Role: models.RoleSudo}

	tok, err := m.Issue(admin, AccessToken)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := m.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.AdminID != 7 || claims.Role != models.RoleSudo || claims.Type != AccessToken {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTWrongSecretRejected(t *testing.T) {
	m1 := NewManager("secret-a", time.Minute, time.Hour)
	m2 := NewManager("secret-b", time.Minute, time.Hour)
	tok, _ := m1.Issue(&models.Admin{ID: 1}, AccessToken)
	if _, err := m2.Parse(tok); err == nil {
		t.Fatal("expected parse failure with wrong secret")
	}
}
