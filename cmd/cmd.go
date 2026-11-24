package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder9527-lg/go-user-service/config"
	"github.com/coder9527-lg/go-user-service/controller"
	"github.com/coder9527-lg/go-user-service/dao"
	"github.com/coder9527-lg/go-user-service/db"
	"github.com/coder9527-lg/go-user-service/model"
	router "github.com/coder9527-lg/go-user-service/routers"
	"github.com/coder9527-lg/go-user-service/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "go-user-service",
	Short: "Gin + GORM 用户服务练习项目",
	RunE:  run,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config/dev.yaml", "config file path")
	rootCmd.PersistentFlags().Bool("debug", true, "enable gin debug mode")
}

// Execute 是命令行入口。
func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	config.MustInit(os.Stdout, cfgFile)

	debug, _ := cmd.Flags().GetBool("debug")
	debug = debug && viper.GetString("gin.mode") != "release"
	// 学习环境打印 SQL，生产环境只记慢查询和错误。
	gormLevel := logger.Warn
	if debug {
		gormLevel = logger.Info
		gin.SetMode(gin.DebugMode)
		logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	} else {
		gin.SetMode(gin.ReleaseMode)
		// 生产环境输出 JSON，便于 ELK/Loki 等日志系统解析与按 request_id 检索。
		logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.000Z07:00"})
	}

	gormDB, err := initDB(gormLevel)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, cerr := gormDB.DB(); cerr == nil {
			if cerr := sqlDB.Close(); cerr != nil {
				logrus.Warnf("close db failed: %v", cerr)
			}
		}
	}()

	// 自动建表，学习环境免手动写 SQL。
	if err := gormDB.AutoMigrate(&model.User{}); err != nil {
		return err
	}

	// 依赖注入组装：gorm.DB → repo → service → handler → router。
	userRepo := dao.NewUserRepo(gormDB)
	userService := service.NewUserService(userRepo)
	userHandler := controller.NewUserHandler(userService)

	srv := &http.Server{
		Addr:              viper.GetString("server.port"),
		Handler:           router.New(userHandler),
		ReadHeaderTimeout: 5 * time.Second, // 防御 Slowloris 慢速攻击
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 监听中断信号，给在途请求 5 秒优雅退出时间。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动失败经 errCh 回传主流程，不能在 goroutine 里用 logrus.Fatalf
	// （它内部 os.Exit 会跳过 defer 的关库与优雅关停）。
	errCh := make(chan error, 1)
	go func() {
		logrus.Infof("server listening on %s", srv.Addr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	select {
	case serveErr := <-errCh:
		return serveErr
	case <-quit:
		logrus.Info("shutting down server ...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

// initDB 按配置选择数据库驱动；默认 SQLite，便于零依赖启动。
func initDB(level logger.LogLevel) (*gorm.DB, error) {
	driver := viper.GetString("db.driver")
	switch driver {
	case "", "sqlite":
		return db.InitSQLiteWithLevel(viper.GetString("db.sqlite_dsn"), level)
	case "mysql":
		gormDB, err := db.InitMySQL(
			viper.GetString("db.hostname"),
			viper.GetInt("db.port"),
			viper.GetString("db.username"),
			viper.GetString("db.password"),
			viper.GetString("db.dbname"),
		)
		if err != nil {
			return nil, err
		}
		db.SetLogLevel(level)
		return gormDB, nil
	default:
		return nil, errors.New("unsupported db driver: " + driver)
	}
}
