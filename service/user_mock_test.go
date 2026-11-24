package service

import (
	"errors"
	"testing"

	"github.com/coder9527-lg/go-user-service/dao"
	"github.com/coder9527-lg/go-user-service/model"
)

// fakeUserRepo 是手写的 UserRepository 测试替身，
// 用内存 map 模拟数据库，整个测试不依赖任何真实 DB 或 SQL。
type fakeUserRepo struct {
	users     map[uint]*model.User
	byPhone   map[string]*model.User
	nextID    uint
	createErr error // 可注入故障，验证 service 的错误处理
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:   make(map[uint]*model.User),
		byPhone: make(map[string]*model.User),
		nextID:  1,
	}
}

func (f *fakeUserRepo) GetByPhone(phone string) (*model.User, error) {
	if u, ok := f.byPhone[phone]; ok {
		return u, nil
	}
	return nil, dao.ErrUserNotFound
}

func (f *fakeUserRepo) GetByID(id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, dao.ErrUserNotFound
}

func (f *fakeUserRepo) Create(u *model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.byPhone[u.Phone]; exists {
		return dao.ErrPhoneDuplicated
	}
	u.ID = f.nextID
	f.nextID++
	// 复制指针，避免外部修改污染存储
	stored := *u
	f.users[u.ID] = &stored
	f.byPhone[u.Phone] = &stored
	return nil
}

// 编译期断言：fake 实现满足 service 要求的接口。
var _ UserRepository = (*fakeUserRepo)(nil)

func TestRegisterWithFakeRepo(t *testing.T) {
	svc := NewUserService(newFakeUserRepo())

	user, err := svc.Register(RegisterInput{Name: "mock", Phone: "13800001111", Password: "secret123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.ID == 0 || user.Password == "secret123" {
		t.Fatal("id should be assigned and password hashed")
	}

	// 重复手机号应被业务层拦截
	if _, err := svc.Register(RegisterInput{Phone: "13800001111", Password: "secret123"}); !errors.Is(err, ErrPhoneRegistered) {
		t.Fatalf("expected ErrPhoneRegistered, got %v", err)
	}
}

func TestRegisterValidationShortCircuitsWithoutTouchingRepo(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewUserService(repo)

	// 非法输入应在校验阶段返回，fake 存储里不应有任何数据。
	if _, err := svc.Register(RegisterInput{Phone: "bad", Password: "secret123"}); !errors.Is(err, ErrInvalidPhone) {
		t.Fatalf("expected ErrInvalidPhone, got %v", err)
	}
	if len(repo.users) != 0 {
		t.Fatal("invalid request must not reach the repository")
	}
}

func TestLoginUnknownPhoneReturnsGenericError(t *testing.T) {
	svc := NewUserService(newFakeUserRepo())

	// 不暴露“用户是否存在”，统一返回凭据错误。
	if _, err := svc.Login(LoginInput{Phone: "13900000000", Password: "x"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterPropagatesInjectedRepoError(t *testing.T) {
	repo := newFakeUserRepo()
	// 模拟数据库故障，验证 service 不会吞掉底层错误。
	repo.createErr = errors.New("disk full")
	svc := NewUserService(repo)

	if _, err := svc.Register(RegisterInput{Phone: "13800002222", Password: "secret123"}); err == nil {
		t.Fatal("expected repository error to propagate")
	}
}
