package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接，由 InitMySQL / InitSQLite 初始化。
var DB *gorm.DB

// SetLogLevel 调整 GORM 日志级别：debug 环境用 Info 打印 SQL，生产用 Warn。
func SetLogLevel(level logger.LogLevel) {
	if DB != nil {
		DB.Logger = DB.Logger.LogMode(level)
	}
}

func open(dialector gorm.Dialector, level logger.LogLevel) (*gorm.DB, error) {
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		// 由调用方按环境指定：debug=Info 打印 SQL，release=Warn 只记慢查询/错误。
		// ErrRecordNotFound 属于正常业务分支（注册查重），不应刷成警告日志。
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel:                  level,
				IgnoreRecordNotFoundError: true,
			},
		),
		// 把驱动原始错误翻译为 gorm 通用错误（如唯一键冲突 → gorm.ErrDuplicatedKey）。
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	// 连接池参数：防止连接数失控和长连接老化。
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = gormDB
	return DB, nil
}

// InitMySQL 连接 MySQL，默认 Warn 日志级别；需要打印 SQL 时调用 SetLogLevel。
func InitMySQL(hostname string, port int, username, password, dbname string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, hostname, port, dbname)
	return open(mysql.Open(dsn), logger.Warn)
}

// InitSQLite 连接 SQLite（纯 Go 驱动，无需 CGO），dsn 传 ":memory:" 即得到内存库。
func InitSQLite(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "go-user-service.db"
	}
	return open(sqlite.Open(dsn), logger.Warn)
}

// InitSQLiteWithLevel 连接 SQLite 并显式指定日志级别，供需要查看 SQL 的本地场景使用。
func InitSQLiteWithLevel(dsn string, level logger.LogLevel) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "go-user-service.db"
	}
	return open(sqlite.Open(dsn), level)
}

// Ping 检查数据库连接是否可用，供就绪探针使用。
func Ping() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Close 关闭底层数据库连接。
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
