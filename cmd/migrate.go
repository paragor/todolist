package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/paragor/todo/pkg/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type MigrateConfig struct {
	SourceConfigPath      string
	DestinationConfigPath string
}

var migrateConfig = MigrateConfig{}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateConfig.SourceConfigPath, "source-config", migrateConfig.SourceConfigPath, "Path to file with with source config")
	migrateCmd.Flags().StringVar(&migrateConfig.DestinationConfigPath, "destination-config", migrateConfig.DestinationConfigPath, "Path to file with with destination config")
}

func loadTaskConfig(cfgPath string) (*Config, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("cant read config file: %w", err)
	}
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("cant unmarshal file: %w", err)
	}
	return config, nil
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate all tasks from one remote instance to another",
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceConfig, err := loadTaskConfig(migrateConfig.SourceConfigPath)
		if err != nil {
			log.Fatalf("cant load source config: %s", err.Error())
		}
		destinationConfig, err := loadTaskConfig(migrateConfig.DestinationConfigPath)
		if err != nil {
			log.Fatalf("cant load destination config: %s", err.Error())
		}

		source := db.NewRemoteRepository(sourceConfig.Client.RemoteAddr, sourceConfig.Client.ServerToken, http.DefaultClient)
		destination := db.NewRemoteRepository(destinationConfig.Client.RemoteAddr, destinationConfig.Client.ServerToken, http.DefaultClient)

		if err := source.Ping(); err != nil {
			log.Fatalf("error on connect to source server: %s", err)
		}
		if err := destination.Ping(); err != nil {
			log.Fatalf("error on connect to destination server: %s", err)
		}

		tasks, err := source.All()
		if err != nil {
			log.Fatalf("cant read tasks from source server: %s", err.Error())
		}
		log.Printf("readed %d tasks\n", len(tasks))
		for _, t := range tasks {
			if err := destination.Insert(t); err != nil {
				log.Fatalf("cant send task to destination server: (%s) %s", t.UUID.String(), err)
			}
		}
		log.Printf("migrated %d tasks\n", len(tasks))
		return nil
	},
}
