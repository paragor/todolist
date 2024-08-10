package models

import (
	"fmt"
	"github.com/google/uuid"
	"slices"
	"strings"
	"time"
	"unicode"
)

const HumanInputHelp = `
NAME
    HumanInputParser - Command-line style parser for task management.

SYNOPSIS
    [action] [UUID] [options...]

DESCRIPTION
    HumanInputParser processes input strings to manage tasks, supporting actions like adding, modifying, listing, and retrieving task information. The input should start with a valid action followed by optional parameters like project, tags, due dates, and notifications.

ACTIONS
    add
        Adds a new task with specified options.

    modify UUID
        Modifies an existing task identified by the given UUID. Requires the task's UUID as the second argument.

    list
        Lists tasks filtered by the specified options.

    info UUID
        Retrieves detailed information about a task identified by the given UUID.

    copy UUID
        Create new task with copy fields from UUID.

    done UUID
        Set status completed for task by the given UUID.

    agenda
        Show tasks that have due today, next 7 day and overdue

OPTIONS
    project:PROJECT_NAME
        Specifies the project name associated with the task. 
        Example: project:MyProject

    status:STATUS
        Sets the task's status. Valid values include pending, completed, deleted.
        Example: status:pending

    +TAG
        Adds a tag to the task. Multiple tags can be added by repeating this option with different tags.
        Example: +urgent +work

    !TAG
        Removes a tag from the task.
        Example: !urgent !work

    due:TIME
        Sets the due date for the task. TIME can be in formats:
		Full datetime:
			2006-01-02T15:04:05Z07:00, 2006-01-02T15:04:05
		Date only (zero for time will be used):
			2006-01-02, 2006.01.02, 02.01.2006
		Time only (today date will be used):
			15:04:05, 15:04
		Relative time from now:
			+1h, -30m
        Example: due:2024-08-20T15:00:00

    notify:TIME
        Sets a notification time for the task. TIME can be in formats:
		Full datetime:
			2006-01-02T15:04:05Z07:00, 2006-01-02T15:04:05
		Date only (zero for time will be used):
			2006-01-02, 2006.01.02, 02.01.2006
		Time only (today date will be used):
			15:04:05, 15:04
		Relative time from now:
			+1h, -30m
        Example: notify:2024-08-15T12:00:00

    ExtraWords...
        Any additional words or phrases will be added to the task's description.
		For list action words used as search words.
        Example: prepare quarterly report

EXAMPLES
    Add a new task with tags and project:
        add +urgent +work project:NewProject prepare presentation

    Modify an existing task by UUID, setting a new due date and description:
        modify 123e4567-e89b-12d3-a456-426614174000 due:2024-08-20 it is new description

    List tasks by project and tag:
        list project:MyProject +urgent

    Retrieve information about a specific task:
        info 123e4567-e89b-12d3-a456-426614174000

    Clone task with new description:
        copy 123e4567-e89b-12d3-a456-426614174000 hue mae
`

type HumanAction string

const (
	HumanActionList   HumanAction = "list"
	HumanActionAdd    HumanAction = "add"
	HumanActionModify HumanAction = "modify"
	HumanActionInfo   HumanAction = "info"
	HumanActionCopy   HumanAction = "copy"
	HumanActionDone   HumanAction = "done"
	HumanActionAgenda HumanAction = "agenda"
)

type HumanInputParserResult struct {
	Action     HumanAction
	ActionUUID *uuid.UUID
	Options    HumanInputOptions
}

type AddOrDeleteValue[T any] struct {
	IsExists bool
	IsAdd    bool
	Value    T
}

type HumanInputOptions struct {
	Project AddOrDeleteValue[string]
	Tags    []AddOrDeleteValue[string]
	Notify  AddOrDeleteValue[time.Time]
	Due     AddOrDeleteValue[time.Time]
	Status  *taskStatus

	ExtraWords []string
}

func (o *HumanInputOptions) ModifyTask(task *Task) {
	if o.Project.IsExists {
		if o.Project.IsAdd {
			task.Project = o.Project.Value
		} else {
			task.Project = ""
		}
	}
	if o.Notify.IsExists {
		if o.Notify.IsAdd {
			notify := o.Notify.Value
			task.Notify = &notify
		} else {
			task.Notify = nil
		}
	}
	if o.Due.IsExists {
		if o.Due.IsAdd {
			due := o.Due.Value
			task.Due = &due
		} else {
			task.Due = nil
		}
	}
	if len(o.ExtraWords) > 0 {
		task.Description = strings.Join(o.ExtraWords, " ")
	}
	if len(o.Tags) > 0 {
		taskTags := map[string]struct{}{}
		for _, tag := range task.Tags {
			taskTags[tag] = struct{}{}
		}
		for _, parsedTag := range o.Tags {
			if !parsedTag.IsExists {
				continue
			}
			if parsedTag.IsAdd {
				taskTags[parsedTag.Value] = struct{}{}
			} else {
				delete(taskTags, parsedTag.Value)
			}
		}
		resultTags := []string{}
		for tag := range taskTags {
			resultTags = append(resultTags, tag)
		}
		slices.Sort(resultTags)
		task.Tags = resultTags
	}
	if o.Status != nil {
		task.Status = *o.Status
	}
}

func (o *HumanInputOptions) ToListFilter() *ListFilter {
	filter := NewDefaultListFilter()
	if o.Project.IsExists && o.Project.IsAdd {
		filter.Project = o.Project.Value
	}
	for _, word := range o.ExtraWords {
		filter.SearchWords = append(filter.SearchWords, strings.ToLower(word))
	}
	for _, tag := range o.Tags {
		if !tag.IsExists || !tag.IsAdd {
			continue
		}
		filter.Tags = append(filter.Tags, tag.Value)
	}

	if o.Status != nil {
		filter.ShowPending = *o.Status == Pending
		filter.ShowCompleted = *o.Status == Completed
		filter.ShowDeleted = *o.Status == Deleted
	}
	return filter
}

func ParseHumanInput(input string) (*HumanInputParserResult, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	firstSpace := strings.IndexFunc(input, unicode.IsSpace)
	if firstSpace < 0 {
		firstSpace = len(input)
	}
	result := &HumanInputParserResult{}
	action := HumanAction(strings.ToLower(input[:firstSpace]))
	allActions := []HumanAction{
		HumanActionAdd, HumanActionModify, HumanActionList,
		HumanActionInfo, HumanActionCopy, HumanActionDone,
		HumanActionAgenda,
	}
	if !slices.Contains(allActions, action) {
		return nil, fmt.Errorf("invalid action: %s", input[:firstSpace])
	}
	result.Action = action
	if firstSpace+1 >= len(input) {
		if action == HumanActionList || action == HumanActionAgenda {
			result.Options = HumanInputOptions{}
			return result, nil
		}
		return nil, fmt.Errorf("only action pass, but option requred")
	}
	input = input[firstSpace+1:]
	input = strings.TrimSpace(input)
	if action == HumanActionModify || action == HumanActionInfo || action == HumanActionCopy || action == HumanActionDone {
		secondSpace := strings.IndexFunc(input, unicode.IsSpace)
		if secondSpace < 0 {
			secondSpace = len(input)
		}
		UUID, err := uuid.Parse(input[:secondSpace])
		if err != nil {
			return nil, fmt.Errorf("for %s action uuid required, cant parse it: %w", action, err)
		}
		result.ActionUUID = &UUID
		input = input[secondSpace:]
	}
	if action == HumanActionDone {
		completedStatus := Completed
		result.Options = HumanInputOptions{
			Status: &completedStatus,
		}
		return result, nil
	}
	if action == HumanActionAgenda {
		result.Options = HumanInputOptions{}
		return result, nil
	}
	options, err := parseHumanOptions(input)
	if err != nil {
		return nil, fmt.Errorf("cant parse options: %w", err)
	}
	result.Options = *options
	return result, nil
}

func parseHumanOptions(input string) (*HumanInputOptions, error) {
	result := &HumanInputOptions{}
	for _, word := range strings.Split(strings.TrimSpace(input), " ") {
		word = strings.TrimSpace(word)
		if len(word) == 0 {
			continue
		}
		if strings.HasPrefix(word, "project:") {
			project := strings.TrimPrefix(word, "project:")
			result.Project = AddOrDeleteValue[string]{IsExists: true, IsAdd: len(project) > 0, Value: project}
			continue
		}
		if strings.HasPrefix(word, "status:") {
			status := strings.TrimPrefix(word, "status:")
			status = strings.ToLower(status)
			if len(status) == 0 {
				return nil, fmt.Errorf("empty status")
			}
			parsedStatus, err := NewTaskStatus(status)
			if err != nil {
				return nil, fmt.Errorf("cant parse status: %w", err)
			}
			result.Status = &parsedStatus
			continue
		}
		if strings.HasPrefix(word, "+") {
			result.Tags = append(result.Tags, AddOrDeleteValue[string]{IsExists: true, IsAdd: true, Value: strings.ToLower(strings.TrimPrefix(word, "+"))})
			continue
		}
		if strings.HasPrefix(word, "!") {
			result.Tags = append(result.Tags, AddOrDeleteValue[string]{IsExists: true, IsAdd: false, Value: strings.ToLower(strings.TrimPrefix(word, "!"))})
			continue
		}
		if strings.HasPrefix(word, "due:") {
			due := strings.TrimPrefix(word, "due:")
			dueValue := AddOrDeleteValue[time.Time]{
				IsExists: true,
				IsAdd:    len(due) > 0,
				Value:    time.Time{},
			}
			if dueValue.IsAdd {
				timeValue, err := parseHumanInputTime(due)
				if err != nil {
					return nil, fmt.Errorf("invalid due: %w", err)
				}
				dueValue.Value = timeValue
			}

			result.Due = dueValue
			continue
		}

		if strings.HasPrefix(word, "notify:") {
			notify := strings.TrimPrefix(word, "notify:")
			notifyValue := AddOrDeleteValue[time.Time]{
				IsExists: true,
				IsAdd:    len(notify) > 0,
				Value:    time.Time{},
			}
			if notifyValue.IsAdd {
				timeValue, err := parseHumanInputTime(notify)
				if err != nil {
					return nil, fmt.Errorf("invalid notify: %w", err)
				}
				notifyValue.Value = timeValue
			}

			result.Notify = notifyValue
			continue
		}

		result.ExtraWords = append(result.ExtraWords, word)
	}

	return result, nil
}

func parseHumanInputTime(word string) (time.Time, error) {
	now := time.Now()
	if t, err := time.Parse(time.RFC3339, word); err == nil {
		return t, nil
	}
	if strings.HasPrefix(word, "+") {
		duration, err := time.ParseDuration(strings.TrimPrefix(word, "+"))
		if err != nil {
			return time.Time{}, fmt.Errorf("cant parse duration: %w", err)
		}
		return time.Now().Add(duration), nil
	}
	if strings.HasPrefix(word, "-") {
		duration, err := time.ParseDuration(strings.TrimPrefix(word, "-"))
		if err != nil {
			return time.Time{}, fmt.Errorf("cant parse duration: %w", err)
		}
		return time.Now().Add(-duration), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", word); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
	}
	if t, err := time.Parse(time.DateOnly, word); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.Parse("2006.01.02", word); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.Parse("02.01.2006", word); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.Parse(time.TimeOnly, word); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
	}
	if t, err := time.Parse("15:04", word); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("cant parse time: %s", strings.Join([]string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		time.DateOnly,
		"2006.01.02",
		"02.01.2006",
		time.TimeOnly,
		"15:04",
		"+1h",
		"-30m",
	}, ", "))
}
