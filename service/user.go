package service

import (
	crand "crypto/rand"
	"errors"
	"regexp"

	"github.com/coder9527-lg/go-user-service/common"
	"github.com/coder9527-lg/go-user-service/dao"
	"github.com/coder9527-lg/go-user-service/model"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository 定义 service 需要的持久化能力。
// 接口定义在消费方（service）而非实现方（dao），是 Go 的惯例：
// service 只依赖抽象，测试时可注入 fake 实现，无需真实数据库。
type UserRepository interface {
	GetByPhone(phone string) (*model.User, error)
	GetByID(id uint) (*model.User, error)
	Create(user *model.User) error
}

// 业务错误：与 HTTP 解耦，由 controller 统一翻译成状态码。
var (
	ErrInvalidPhone       = errors.New("phone must be 11 digits starting with 1")
	ErrInvalidPassword    = errors.New("password length must be between 6 and 72 characters")
	ErrPhoneRegistered    = errors.New("phone already registered")
	ErrInvalidCredentials = errors.New("incorrect phone or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
)

// 中国大陆手机号：1 开头、共 11 位数字。
var phoneRegexp = regexp.MustCompile(`^1\d{10}$`)

const (
	nameLetters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	minPasswordLength = 6
	// bcrypt 对超过 72 字节的密码会直接报错，这里提前拦截并给出友好提示。
	maxPasswordLength = 72
)

// RegisterInput 注册入参，纯业务结构，不含任何 HTTP 标签。
type RegisterInput struct {
	Name     string
	Phone    string
	Password string
}

// LoginInput 登录入参。
type LoginInput struct {
	Phone    string
	Password string
}

// UserService 承载用户相关的业务规则与流程编排。
type UserService struct {
	repo UserRepository
}

// NewUserService 通过构造函数注入仓储依赖。
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Register 注册流程：参数校验 → 查重 → 密码哈希 → 落库。
func (s *UserService) Register(in RegisterInput) (*model.User, error) {
	if !phoneRegexp.MatchString(in.Phone) {
		return nil, ErrInvalidPhone
	}
	if len(in.Password) < minPasswordLength || len(in.Password) > maxPasswordLength {
		return nil, ErrInvalidPassword
	}
	name := in.Name
	if name == "" {
		name = randomName(10)
	}

	// 手机号唯一校验：唯一索引之外再加一层友好提示。
	existing, err := s.repo.GetByPhone(in.Phone)
	if err != nil && !errors.Is(err, dao.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPhoneRegistered
	}

	// 密码使用 bcrypt 哈希存储，绝不保存明文，也不打印到日志。
	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Name:     name,
		Phone:    in.Phone,
		Password: string(hashed),
	}
	if err := s.repo.Create(user); err != nil {
		// 并发注册时唯一索引兜底：把 dao 层错误翻译成业务错误。
		if errors.Is(err, dao.ErrPhoneDuplicated) {
			return nil, ErrPhoneRegistered
		}
		return nil, err
	}
	return user, nil
}

// Login 登录流程：查用户 → 校验密码哈希 → 签发 access/refresh 双令牌。
// 用户不存在与密码错误返回同一个错误，避免被枚举手机号。
func (s *UserService) Login(in LoginInput) (*common.TokenPair, error) {
	user, err := s.repo.GetByPhone(in.Phone)
	if errors.Is(err, dao.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	pair, err := common.IssueTokenPair(*user)
	if err != nil {
		return nil, err
	}
	return pair, nil
}

// Refresh 用 refresh token 换取新的 access token。
func (s *UserService) Refresh(refreshToken string) (string, error) {
	access, err := common.RefreshAccessToken(refreshToken)
	if err != nil {
		return "", ErrInvalidToken
	}
	return access, nil
}

// GetByID 按主键查询用户，供 /me 等需要当前用户信息的场景使用。
func (s *UserService) GetByID(id uint) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if errors.Is(err, dao.ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// randomName 使用 crypto/rand 生成随机默认昵称。
func randomName(n int) string {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "user"
	}
	for i := range b {
		b[i] = nameLetters[int(b[i])%len(nameLetters)]
	}
	return string(b)
}
