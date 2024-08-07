package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/paragor/todo/pkg/db"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run todolist console client",
	Run: func(cmd *cobra.Command, args []string) {
		repo := db.NewRemoteRepository(cfg.Client.RemoteAddr, cfg.Client.ServerToken, http.DefaultClient)
		if err := repo.Ping(); err != nil {
			log.Fatalf("cant connect to server: %s", err.Error())
		}
		tasks, err := repo.All()
		if err != nil {
			log.Fatalf("cant get tasks: %s", err.Error())
		}
		data, err := json.MarshalIndent(tasks, "", "  ")
		if err != nil {
			log.Fatalf("cant marshal tasks: %s", err.Error())

		}
		fmt.Print(string(data))
	},
}
