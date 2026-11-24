package dao

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/coder9527-lg/go-user-service/db"
	"github.com/coder9527-lg/go-user-service/model"
)

// 每个测试用例分配一个独立命名的共享内存库：
// 同一用例内连接池的多条连接共享数据（含并发场景），不同用例之间又互不干扰。
var dbSeq uint64

func sharedMemoryDSN(t *testing.T) string {
	t.Helper()
	n := atomic.AddUint64(&dbSeq, 1)
	return fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", n)
}

// newTestRepo 初始化独立内存库并返回基于它的 UserRepo。
func newTestRepo(t *testing.T) *UserRepo {
	t.Helper()
	gormDB, err := db.InitSQLite(sharedMemoryDSN(t))
	if err != nil {
		t.Fatalf("init sqlite: %v", err)
	}
	if err := gormDB.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return NewUserRepo(gormDB)
}

func TestCreateAndGetUserByPhone(t *testing.T) {
	repo := newTestRepo(t)

	user := &model.User{Name: "alice", Phone: "13800000001", Password: "hashed-pwd"}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be populated")
	}

	got, err := repo.GetByPhone("13800000001")
	if err != nil {
		t.Fatalf("GetByPhone failed: %v", err)
	}
	if got.Name != "alice" || got.Phone != "13800000001" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestGetUserByPhoneNotFound(t *testing.T) {
	repo := newTestRepo(t)

	if _, err := repo.GetByPhone("13900000009"); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	repo := newTestRepo(t)

	user := &model.User{Name: "erin", Phone: "13600000001", Password: "hashed-pwd"}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Phone != "13600000001" {
		t.Fatalf("unexpected user: %+v", got)
	}

	if _, err := repo.GetByID(99999); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestPhoneUniqueIndex(t *testing.T) {
	repo := newTestRepo(t)

	u1 := &model.User{Name: "bob", Phone: "13700000001", Password: "hashed-pwd"}
	u2 := &model.User{Name: "bob2", Phone: "13700000001", Password: "hashed-pwd"}
	if err := repo.Create(u1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := repo.Create(u2); err == nil {
		t.Fatal("expected unique constraint error for duplicate phone")
	}
}
