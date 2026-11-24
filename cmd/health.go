package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// healthCmd 供容器 HEALTHCHECK 调用：访问存活探针，非 200 即以非零码退出。
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "检查服务存活探针，供容器健康检查使用",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + addr + "/ping")
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return errors.New("unexpected status: " + resp.Status)
		}
		fmt.Println("ok")
		return nil
	},
}

func init() {
	healthCmd.Flags().String("addr", "127.0.0.1:8080", "服务监听地址")
	rootCmd.AddCommand(healthCmd)
}
