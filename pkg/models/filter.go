package models

import (
	"slices"
	"strings"
)

func NewDefaultListFilter() *ListFilter {
	return &ListFilter{
		ShowPending:   true,
		ShowDeleted:   false,
		ShowCompleted: false,
		Tags:          nil,
		SearchWords:   nil,
		Project:       "",
	}
}

type ListFilter struct {
	ShowPending   bool
	ShowDeleted   bool
	ShowCompleted bool
	Tags          []string
	SearchWords   []string
	Project       string
}

func (filter *ListFilter) Apply(tasks []*Task) []*Task {
	return slices.DeleteFunc(tasks, func(task *Task) bool {
		if !filter.ShowDeleted && task.Status == Deleted {
			return true
		}
		if !filter.ShowCompleted && task.Status == Completed {
			return true
		}
		if !filter.ShowPending && task.Status == Pending {
			return true
		}

		if len(filter.Project) > 0 {
			if filter.Project == ProjectSelectorEmpty {
				if task.Project != "" {
					return true
				}
			} else if strings.ToLower(task.Project) != strings.ToLower(filter.Project) {
				return true
			}
		}

		if len(filter.Tags) > 0 {
			whitelistTags := slices.Clone(filter.Tags)
			for i := range whitelistTags {
				whitelistTags[i] = strings.ToLower(whitelistTags[i])
			}

			if slices.Contains(whitelistTags, "project") {
				whitelistTags = slices.DeleteFunc(whitelistTags, func(s string) bool {
					return s == "project"
				})
				if len(task.Project) == 0 {
					return true
				}
			}

			foundTags := 0
			for _, wtag := range whitelistTags {
				wtag = strings.ToLower(wtag)
				if slices.ContainsFunc(task.Tags, func(s string) bool {
					return wtag == strings.ToLower(s)
				}) {
					foundTags++
				}
			}
			if foundTags != len(whitelistTags) {
				return true
			}
		}
		if len(filter.SearchWords) > 0 {
			foundWords := 0
			for _, w := range filter.SearchWords {
				w = strings.ToLower(w)
				if strings.Contains(strings.ToLower(task.Description), w) {
					foundWords++
				}
			}
			if foundWords != len(filter.SearchWords) {
				return true
			}
		}
		return false
	})
}
