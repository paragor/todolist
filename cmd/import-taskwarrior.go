package cmd

import (
	"github.com/paragor/todo/pkg/db"
	"github.com/paragor/todo/pkg/taskwarrior"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var taskwarriorImportConfig = &taskwarrior.ImportConfig{
	Filepath:      "",
	SkipDeleted:   true,
	SkipCompleted: true,
	SkipRecur:     true,
}

func init() {
	rootCmd.AddCommand(importTaskWarriorCmd)
	importTaskWarriorCmd.Flags().StringVar(&taskwarriorImportConfig.Filepath, "tasks-export-file", taskwarriorImportConfig.Filepath, "Path to file with stdout of eval 'task export'. If not set - 'task export' will evaluated")
	importTaskWarriorCmd.Flags().BoolVar(&taskwarriorImportConfig.SkipCompleted, "skip-completed", taskwarriorImportConfig.SkipCompleted, "skip completed tasks")
	importTaskWarriorCmd.Flags().BoolVar(&taskwarriorImportConfig.SkipDeleted, "skip-deleted", taskwarriorImportConfig.SkipDeleted, "skip deleted tasks")
	importTaskWarriorCmd.Flags().BoolVar(&taskwarriorImportConfig.SkipRecur, "skip-recur", taskwarriorImportConfig.SkipRecur, "skip recur tasks")
}

var importTaskWarriorCmd = &cobra.Command{
	Use:   "import-taskwarrior",
	Short: "import tasks from taskwarrior",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := db.NewRemoteRepository(cfg.Client.RemoteAddr, cfg.Client.ServerToken, http.DefaultClient)
		if err := repo.Ping(); err != nil {
			log.Fatalf("error on connect to remote server: %s", err)
		}

		tasks, err := taskwarrior.Import(taskwarriorImportConfig)
		if err != nil {
			log.Fatalf("error on export tasks: %s", err)
		}

		for _, t := range tasks {
			if err := repo.Insert(t); err != nil {
				log.Fatalf("cant import task: (%s) %s", t.UUID.String(), err)
			}
		}

		log.Printf("imported %d tasks\n", len(tasks))
		return nil
	},
}
