package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path"
	"strings"
)

var homeDir = path.Join(Or(os.Getenv("TODOLIST_HOME"), os.Getenv("HOME")), ".config/todolist")

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", cfgFile, "config file (env TODOLIST_CONFIG_PATH)")
}

var rootCmd = &cobra.Command{
	Use:   "todolist",
	Short: "Todolist is todo list :) https://github.com/paragor/todolist",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if strings.Contains(cfgFile, homeDir) {
		if err := os.MkdirAll(homeDir, 0755); err != nil {
			panic(fmt.Errorf("cant init home dir: %w", err).Error())
		}
	}
	f, err := os.Open(cfgFile)
	if err != nil && os.IsNotExist(err) {
		return
	} else if err != nil {
		panic(fmt.Errorf("cant open config file: %w", err).Error())
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		panic(fmt.Errorf("cant read config file: %w", err).Error())
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		panic(fmt.Errorf("cant unmarshal config file: %w", err).Error())
	}
}

func Or[T comparable](value T, alternatives ...T) T {
	var zero T
	if value != zero {
		return value
	}
	for _, alternative := range alternatives {
		if alternative != zero {
			return alternative
		}
	}

	return zero
}
