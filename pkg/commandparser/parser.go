package commandparser

import (
	"fmt"
	"strings"
	"time"
)

type AddOrDeleteValue[T any] struct {
	IsExists bool
	IsAdd    bool
	Value    T
}

type ParserResult struct {
	Action  string
	Options map[string]string
	Project AddOrDeleteValue[string]
	Tags    []AddOrDeleteValue[string]
	Notify  AddOrDeleteValue[time.Time]
	Due     AddOrDeleteValue[time.Time]
	Status  *string

	ExtraWords []string
}

func ParseCommand(command string) (*ParserResult, error) {
	result := &ParserResult{
		Options: map[string]string{},
	}
	actionFound := false
	for _, word := range strings.Split(strings.TrimSpace(command), " ") {
		word = strings.TrimSpace(word)
		if len(word) == 0 {
			continue
		}
		if !actionFound {
			actionFound = true
			result.Action = word
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
			result.Status = &status
			continue
		}
		if strings.HasPrefix(word, "+") {
			result.Tags = append(result.Tags, AddOrDeleteValue[string]{IsExists: true, IsAdd: true, Value: strings.TrimPrefix(word, "+")})
			continue
		}
		if strings.HasPrefix(word, "-") {
			result.Tags = append(result.Tags, AddOrDeleteValue[string]{IsExists: true, IsAdd: false, Value: strings.TrimPrefix(word, "-")})
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
				timeValue, err := parseTime(due)
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
				timeValue, err := parseTime(notify)
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
	if !actionFound {
		return nil, fmt.Errorf("no action pass")
	}

	return result, nil
}

func parseTime(word string) (time.Time, error) {
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
