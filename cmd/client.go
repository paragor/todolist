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
	RunE: func(cmd *cobra.Command, args []string) error {
		if clientOutput != "json" && clientOutput != "table" {
			return fmt.Errorf("unknown output format")
		}
		input := strings.Join(args, " ")
		strings.TrimSpace(input)
		if len(input) == 0 {
			return fmt.Errorf("empty input")
		}
		repo := db.NewRemoteRepository(cfg.Client.RemoteAddr, cfg.Client.ServerToken, http.DefaultClient)
		if err := repo.Ping(); err != nil {
			log.Fatalf("cant connect to server: %s", err.Error())
		}

		parsedInput, err := models.ParseHumanInput(input)
		if err != nil {
			log.Fatalf("cant parse command: %s", err.Error())
		}
		switch parsedInput.Action {
		case models.HumanActionInfo:
			task, err := repo.Get(*parsedInput.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			outputTasks([]*models.Task{task})
			return nil
		case models.HumanActionAdd:
			task := models.NewTask()
			parsedInput.Options.ModifyTask(task)
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			outputTasks([]*models.Task{task})
			return nil
		case models.HumanActionModify, models.HumanActionDone:
			task, err := repo.Get(*parsedInput.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			parsedInput.Options.ModifyTask(task)
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			outputTasks([]*models.Task{task})
			return nil
		case models.HumanActionCopy:
			task, err := repo.Get(*parsedInput.ActionUUID)
			if err != nil {
				log.Fatalf("cant fetch task: %s", err.Error())
			}
			parsedInput.Options.ModifyTask(task)
			task = task.Clone(true)
			if task.Status != models.Pending && parsedInput.Options.Status == nil {
				task.Status = models.Pending
			}
			if err := repo.Insert(task); err != nil {
				log.Fatalf("cant insert task: %s", err.Error())
			}
			outputTasks([]*models.Task{task})
			return nil
		case models.HumanActionList:
			tasks, err := repo.All()
			if err != nil {
				log.Fatalf("cant get tasks: %s", err.Error())
			}
			tasks = parsedInput.Options.ToListFilter().Apply(tasks)
			outputTasks(tasks)
			return nil
		case models.HumanActionAgenda:
			tasks, err := repo.All()
			if err != nil {
				log.Fatalf("cant get tasks: %s", err.Error())
			}
			tasks = models.NewDefaultListFilter().Apply(tasks)
			outputAgenda(models.Agenda(tasks))
			return nil
		default:
			log.Fatalf("unkown action: %s", parsedInput.Action)
		}
		return nil
	},
}

func outputAgenda(agenda []models.TaskGroup) {
	if clientOutput == "json" {
		fmt.Println(prettyOutputJson(agenda))
	} else {
		fmt.Println(prettyOutputTasksGroupsTable(agenda))
	}
}

func outputTasks(tasks []*models.Task) {
	if clientOutput == "json" {
		fmt.Println(prettyOutputJson(tasks))
	} else {
		fmt.Println(prettyOutputTasksTable(tasks))
	}
}
func prettyOutputTasksGroupsTable(groups []models.TaskGroup) string {
	tableWriter := table.NewWriter()
	headerRow := tableGetTasksHeaderRow()
	tableWriter.AppendHeader(headerRow)
	for i, group := range groups {
		if i > 0 {
			tableWriter.AppendSeparator()
		}
		groupNameRow := table.Row{group.Group}
		for i := 1; i < len(headerRow); i++ {
			groupNameRow = append(groupNameRow, "")
		}
		tableWriter.AppendRow(groupNameRow, table.RowConfig{AutoMerge: true})
		tableWriter.AppendSeparator()
		tableWriter.AppendRows(tableGetTasksBodyRows(group.Tasks))
	}
	return tableWriter.Render()
}

func prettyOutputTasksTable(tasks []*models.Task) string {
	tableWriter := table.NewWriter()
	tableWriter.AppendHeader(tableGetTasksHeaderRow())
	tableWriter.AppendRows(tableGetTasksBodyRows(tasks))
	return tableWriter.Render()
}

func tableGetTasksHeaderRow() table.Row {
	return table.Row{"uuid", "status", "project", "tags", "description", "due", "notify"}
}

func tableGetTasksBodyRows(tasks []*models.Task) []table.Row {
	mbDate := func(date *time.Time) string {
		if date == nil {
			return ""
		} else {
			return date.In(time.Local).Format("2006-01-02 15:04")
		}
	}

	result := []table.Row{}
	for _, task := range tasks {
		result = append(result, table.Row{
			task.UUID.String(),
			task.Status,
			task.Project,
			strings.Join(task.Tags, ", "),
			task.Description,
			mbDate(task.Due),
			mbDate(task.Notify),
		})
	}
	return result
}

func prettyOutputJson(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatalf("cant marshal tasks: %s", err.Error())
	}
	return string(data)
}
