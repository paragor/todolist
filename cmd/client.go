package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/paragor/todo/pkg/db"
	"github.com/paragor/todo/pkg/models"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"strings"
	"time"
)

var clientOutput = "table"

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.Flags().StringVarP(&clientOutput, "output", "o", clientOutput, "output format (json, table)")
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run todolist console client",
	Long:  models.HumanInputHelp,
	Run: func(cmd *cobra.Command, args []string) {
		if clientOutput != "json" && clientOutput != "table" {
			log.Fatalln("unknown output format")
		}
		repo := db.NewRemoteRepository(cfg.Client.RemoteAddr, cfg.Client.ServerToken, http.DefaultClient)
		if err := repo.Ping(); err != nil {
			log.Fatalf("cant connect to server: %s", err.Error())
		}

		input, err := models.ParseHumanInput(strings.Join(args, " "))
		if err != nil {
			log.Fatalf("cant parse command: %s", err.Error())
		}
		var result []*models.Task
		switch input.Action {
		case models.HumanActionInfo:
			task, err := repo.Get(*input.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			result = []*models.Task{task}
		case models.HumanActionAdd:
			task := models.NewTask()
			input.Options.ModifyTask(task)
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			result = []*models.Task{task}
		case models.HumanActionModify, models.HumanActionDone:
			task, err := repo.Get(*input.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			input.Options.ModifyTask(task)
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			result = []*models.Task{task}
		case models.HumanActionCopy:
			task, err := repo.Get(*input.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			input.Options.ModifyTask(task)
			task = task.Clone(true)
			if task.Status != models.Pending && input.Options.Status == nil {
				task.Status = models.Pending
			}
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			result = []*models.Task{task}
		case models.HumanActionList:
			tasks, err := repo.All()
			if err != nil {
				log.Fatalf("cant get tasks: %s", err.Error())
			}
			tasks = input.Options.ToListFilter().Apply(tasks)
			result = tasks
		default:
			log.Fatalf("unkown action: %s", input.Action)
		}

		if clientOutput == "json" {
			fmt.Println(prettyOutputJson(result))
		} else {
			fmt.Println(prettyOutputTable(result))
		}
	},
}

func prettyOutputTable(tasks []*models.Task) string {
	mbDate := func(date *time.Time) string {
		if date == nil {
			return ""
		} else {
			return date.In(time.Local).Format("2006-01-02 15:04")
		}
	}
	tableWriter := table.NewWriter()
	tableWriter.AppendHeader(table.Row{"uuid", "project", "description", "status", "tags", "due", "notify"})
	for _, task := range tasks {
		tableWriter.AppendRow(table.Row{
			task.UUID.String(),
			task.Project,
			task.Description,
			task.Status,
			strings.Join(task.Tags, ", "),
			mbDate(task.Due),
			mbDate(task.Notify),
		})
	}
	return tableWriter.Render()
}

func prettyOutputJson(tasks []*models.Task) string {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		log.Fatalf("cant marshal tasks: %s", err.Error())
	}
	return string(data)
}
