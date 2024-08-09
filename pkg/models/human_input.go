package models

import (
	"fmt"
	"github.com/google/uuid"
	"slices"
	"strings"
	"time"
	"unicode"
)

type HumanAction string

const (
	HumanActionList   HumanAction = "list"
	HumanActionAdd    HumanAction = "add"
	HumanActionModify HumanAction = "modify"
	HumanActionInfo   HumanAction = "info"
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
		filter.ShowPending = *o.Status == Deleted
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
	var action HumanAction
	switch strings.ToLower(input[:firstSpace]) {
	case string(HumanActionAdd):
		action = HumanActionAdd
	case string(HumanActionModify):
		action = HumanActionModify
	case string(HumanActionList):
		action = HumanActionList
	case string(HumanActionInfo):
		action = HumanActionInfo
	default:
		return nil, fmt.Errorf("invalid action: %s", input[:firstSpace])
	}
	result.Action = action
	if firstSpace+1 >= len(input) {
		if action == HumanActionList {
			result.Options = HumanInputOptions{}
			return result, nil
		}
		return nil, fmt.Errorf("only action pass, but option requred")
	}
	input = input[firstSpace+1:]
	input = strings.TrimSpace(input)
	if action == HumanActionModify || action == HumanActionInfo {
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
		if strings.HasPrefix(word, "-") {
			result.Tags = append(result.Tags, AddOrDeleteValue[string]{IsExists: true, IsAdd: false, Value: strings.ToLower(strings.TrimPrefix(word, "-"))})
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
	}, ", "))
}
