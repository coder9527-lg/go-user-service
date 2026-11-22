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

// setupTestDB 每个测试用例使用独立的内存 SQLite。
func setupTestDB(t *testing.T) {
	t.Helper()
	gormDB, err := db.InitSQLite(sharedMemoryDSN(t))
	if err != nil {
		t.Fatalf("init sqlite: %v", err)
	}
	if err := gormDB.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
}

func TestCreateAndGetUserByPhone(t *testing.T) {
	setupTestDB(t)

	user := &model.User{Name: "alice", Phone: "13800000001", Password: "hashed-pwd"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be populated")
	}

	got, err := GetUserByPhone("13800000001")
	if err != nil {
		t.Fatalf("GetUserByPhone failed: %v", err)
	}
	if got.Name != "alice" || got.Phone != "13800000001" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestGetUserByPhoneNotFound(t *testing.T) {
	setupTestDB(t)

	if _, err := GetUserByPhone("13900000009"); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	setupTestDB(t)

	user := &model.User{Name: "erin", Phone: "13600000001", Password: "hashed-pwd"}
	if err := CreateUser(user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	got, err := GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.Phone != "13600000001" {
		t.Fatalf("unexpected user: %+v", got)
	}

	if _, err := GetUserByID(99999); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestPhoneUniqueIndex(t *testing.T) {
	setupTestDB(t)

	u1 := &model.User{Name: "bob", Phone: "13700000001", Password: "hashed-pwd"}
	u2 := &model.User{Name: "bob2", Phone: "13700000001", Password: "hashed-pwd"}
	if err := CreateUser(u1); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}
	if err := CreateUser(u2); err == nil {
		t.Fatal("expected unique constraint error for duplicate phone")
	}
}
