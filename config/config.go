package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	configType = "yaml"
)

func Init(output io.Writer, configFile string) error {
	if output == nil {
		output = io.Discard
	}

	viper.SetConfigFile(configFile)
	viper.SetConfigType(configType)

	// 允许环境变量覆盖配置：viper key 中的 "." 和 "-" 映射为 "_"。
	// 例如 db.password 对应 DB_PASSWORD，jwt.secret 对应 JWT_SECRET。
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		return err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		_, _ = fmt.Fprintf(output, "Config file changed %s \n", e.Name)
	})
	return nil
}

func MustInit(output io.Writer, conf string) {
	if err := Init(output, conf); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
