package models

import (
	"fmt"
	"github.com/google/uuid"
	"html"
	"html/template"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

type Task struct {
	UUID        uuid.UUID  `json:"uuid"`
	Description string     `json:"description"`
	Project     string     `json:"project,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Status      taskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	Due         *time.Time `json:"due,omitempty"`
	Notify      *time.Time `json:"notify,omitempty"`
}

const ProjectSelectorEmpty = "__empty__"

func NewTaskStatus(status string) (taskStatus, error) {
	switch status {
	case "completed":
		return Completed, nil
	case "deleted":
		return Deleted, nil
	case "pending":
		return Pending, nil
	}
	return invalid, fmt.Errorf("invalid status")
}

type taskStatus string

func (ts *taskStatus) String() string {
	return string(*ts)
}

func (ts *taskStatus) Emoji() string {
	switch string(*ts) {
	case "completed":
		return "✅"
	case "deleted":
		return "🗑️"
	case "pending":
		return "⏳"
	}
	return string(*ts)
}

func (ts *taskStatus) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("expected string")
	}
	data = data[1 : len(data)-1]
	switch string(data) {
	case string(Pending):
		*ts = Pending
		return nil
	case string(Completed):
		*ts = Completed
		return nil
	case string(Deleted):
		*ts = Deleted
		return nil
	default:
		return fmt.Errorf("unknown status: %s", string(data))
	}
}

const (
	Pending   taskStatus = "pending"
	Completed taskStatus = "completed"
	Deleted   taskStatus = "deleted"

	invalid taskStatus = "invalid"
)

var linkRegexp = regexp.MustCompile("(https?://([^/]+)\\S*)")

func (t *Task) HtmlDescription() template.HTML {
	result := linkRegexp.ReplaceAllString(html.EscapeString(t.Description), `<a href="$1" target="_blank">$2</a>`)
	return template.HTML(result)
}

func (t *Task) Validate() error {
	if t.UUID == uuid.Nil {
		return fmt.Errorf("uuid should not be nil")
	}
	if len(t.Description) == 0 {
		return fmt.Errorf("description should not be empty")
	}
	if t.Status != Pending && t.Status != Deleted && t.Status != Completed {
		return fmt.Errorf("unknown status: %s", t.Status)
	}
	if t.CreatedAt.IsZero() {
		return fmt.Errorf("created at should not be zero")
	}

	return nil
}

func (t *Task) Unify() {
	t.Project = strings.TrimSpace(strings.ToLower(t.Project))

	tags := []string{}
	for i := range t.Tags {
		tags = append(tags, strings.TrimSpace(strings.ToLower(t.Tags[i])))
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i] < tags[j]
	})
	tags = slices.DeleteFunc(tags, func(s string) bool {
		return s == ""
	})
	tags = slices.Compact(tags)
	t.Tags = tags
	if t.Due != nil && t.Notify == nil {
		notify := *t.Due
		t.Notify = &notify
	}
}

func (t *Task) Clone(newUuid bool) *Task {
	UUID := t.UUID
	if newUuid {
		UUID = uuid.New()
	}
	tags := []string{}
	for _, t := range t.Tags {
		tags = append(tags, t)
	}
	return &Task{
		UUID:        UUID,
		Description: t.Description,
		Project:     t.Project,
		Status:      t.Status,
		Tags:        tags,
		CreatedAt:   t.CreatedAt,
		Due:         t.Due,
		Notify:      t.Notify,
	}
}

type TaskGroup struct {
	Group string
	Tasks []*Task
}

func GroupTasksByProject(tasks []*Task) []TaskGroup {
	result := map[string][]*Task{}
	for _, t := range tasks {
		project := ProjectSelectorEmpty
		if len(t.Project) > 0 {
			project = t.Project
		}
		result[project] = append(result[t.Project], t)
	}
	groups := []TaskGroup{}
	for g, ts := range result {
		SortTasks(ts)
		groups = append(groups, TaskGroup{g, ts})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Group == ProjectSelectorEmpty {
			return true
		}
		return groups[i].Group < groups[j].Group
	})

	return groups
}

func compareStatus(status taskStatus) int {
	switch status {
	case Pending:
		return 0
	case Completed:
		return 1
	case Deleted:
		return -1

	}
	panic("invalid status")
}

func SortTasks(tasks []*Task) {
	slices.SortFunc(tasks, func(a, b *Task) int {
		if compareStatus(a.Status) < compareStatus(b.Status) {
			return -1
		}
		if compareStatus(a.Status) > compareStatus(b.Status) {
			return 1
		}
		if a.Due != nil && b.Due != nil {
			if a.Due.Before(*b.Due) {
				return -1
			}
			if b.Due.Before(*a.Due) {
				return 1
			}
		} else if a.Due != nil {
			return -1
		} else if b.Due != nil {
			return 1
		}
		if a.Project < b.Project {
			return -1
		}
		if a.Project > b.Project {
			return 1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if b.CreatedAt.Before(a.CreatedAt) {
			return 1
		}
		if a.UUID.String() < b.UUID.String() {
			return -1
		}
		if a.UUID.String() > b.UUID.String() {
			return 1
		}
		return 0
	})
}

func UniqProjects(tasks []*Task) map[string]int {
	uniqItems := map[string]int{}
	for _, t := range tasks {
		project := t.Project
		if project == "" {
			project = ProjectSelectorEmpty
		}
		uniqItems[project]++
	}
	return uniqItems
}

func UniqTags(tasks []*Task) map[string]int {
	uniqItems := map[string]int{}
	for _, t := range tasks {
		for _, tag := range t.Tags {
			uniqItems[tag]++
		}
	}
	return uniqItems
}
