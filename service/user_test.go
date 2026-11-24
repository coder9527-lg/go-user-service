package service

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coder9527-lg/go-user-service/dao"
	"github.com/coder9527-lg/go-user-service/db"
	"github.com/coder9527-lg/go-user-service/model"
	"golang.org/x/crypto/bcrypt"
)

var seq uint64

// setupServiceDB 为每个用例准备独立的共享内存库，并组装真实 repo + service。
func setupServiceDB(t *testing.T) *UserService {
	t.Helper()
	n := atomic.AddUint64(&seq, 1)
	gormDB, err := db.InitSQLite("file:servicetest" + strconv.FormatUint(n, 10) + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := gormDB.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	repo := dao.NewUserRepo(gormDB)
	return NewUserService(repo)
}

func TestRegisterSuccessAndHashedPassword(t *testing.T) {
	s := setupServiceDB(t)

	user, err := s.Register(RegisterInput{Name: "alice", Phone: "13800000001", Password: "secret123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.ID == 0 || user.Password == "secret123" {
		t.Fatal("password must be hashed before persistence")
	}
	// 哈希应能通过 bcrypt 校验。
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("secret123")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
}

func TestRegisterAssignsRandomNameWhenEmpty(t *testing.T) {
	s := setupServiceDB(t)

	user, err := s.Register(RegisterInput{Phone: "13800000002", Password: "secret123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.Name == "" {
		t.Fatal("expected an auto-generated name")
	}
}

func TestRegisterValidationErrors(t *testing.T) {
	s := setupServiceDB(t)

	cases := []struct {
		name string
		in   RegisterInput
		want error
	}{
		{"bad phone", RegisterInput{Phone: "123", Password: "secret123"}, ErrInvalidPhone},
		{"short password", RegisterInput{Phone: "13800000003", Password: "123"}, ErrInvalidPassword},
		{"overlong password", RegisterInput{Phone: "13800000004", Password: strings.Repeat("a", 73)}, ErrInvalidPassword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Register(tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestRegisterDuplicatePhone(t *testing.T) {
	s := setupServiceDB(t)
	in := RegisterInput{Phone: "13800000005", Password: "secret123"}

	if _, err := s.Register(in); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if _, err := s.Register(in); !errors.Is(err, ErrPhoneRegistered) {
		t.Fatalf("expected ErrPhoneRegistered, got %v", err)
	}
}

func TestLogin(t *testing.T) {
	s := setupServiceDB(t)
	if _, err := s.Register(RegisterInput{Phone: "13800000006", Password: "secret123"}); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	t.Run("correct password returns token pair", func(t *testing.T) {
		pair, err := s.Login(LoginInput{Phone: "13800000006", Password: "secret123"})
		if err != nil || pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatalf("expected token pair, got err=%v pair=%+v", err, pair)
		}
		if pair.ExpiresIn != 7200 {
			t.Fatalf("expected access ttl 7200s, got %d", pair.ExpiresIn)
		}
	})

	t.Run("wrong password returns generic error", func(t *testing.T) {
		if _, err := s.Login(LoginInput{Phone: "13800000006", Password: "wrong"}); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown phone returns same generic error", func(t *testing.T) {
		if _, err := s.Login(LoginInput{Phone: "13900000009", Password: "secret123"}); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}

func TestGetByID(t *testing.T) {
	s := setupServiceDB(t)
	user, err := s.Register(RegisterInput{Phone: "13800000007", Password: "secret123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	got, err := s.GetByID(user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Phone != "13800000007" {
		t.Fatalf("unexpected user: %+v", got)
	}

	if _, err := s.GetByID(99999); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	s := setupServiceDB(t)
	if _, err := s.Register(RegisterInput{Phone: "13800000008", Password: "secret123"}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	pair, err := s.Login(LoginInput{Phone: "13800000008", Password: "secret123"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	t.Run("valid refresh token issues new access token", func(t *testing.T) {
		access, err := s.Refresh(pair.RefreshToken)
		if err != nil || access == "" {
			t.Fatalf("expected new access token, got err=%v token=%q", err, access)
		}
		if access == pair.AccessToken {
			t.Fatal("refreshed access token should differ from the original")
		}
	})

	t.Run("access token cannot be used to refresh", func(t *testing.T) {
		if _, err := s.Refresh(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("garbage token is rejected", func(t *testing.T) {
		if _, err := s.Refresh("not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})
}
