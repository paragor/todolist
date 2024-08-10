package httpserver

import (
	"github.com/paragor/todo/pkg/models"
	"net/url"
	"strings"
)

func ListFilterToQuery(filter *models.ListFilter) url.Values {
	query := url.Values{}
	if filter.ShowDeleted {
		query.Add("show_deleted", "true")
	}
	if filter.ShowCompleted {
		query.Add("show_completed", "true")
	}
	if filter.Project != "" {
		query.Add("project", filter.Project)
	}
	for _, tag := range filter.Tags {
		query.Add("tags", tag)
	}
	for _, word := range filter.SearchWords {
		query.Add("search_words", word)
	}
	return query
}

func queryToListFilter(query url.Values) *models.ListFilter {
	if query.Has("all") {
		return &models.ListFilter{
			ShowPending:   true,
			ShowDeleted:   true,
			ShowCompleted: true,
			Tags:          nil,
			SearchWords:   nil,
			Project:       "",
		}
	}
	filter := &models.ListFilter{
		ShowDeleted:   query.Has("show_deleted"),
		ShowCompleted: query.Has("show_completed"),
		ShowPending:   true,
		Tags:          nil,
		SearchWords:   nil,
		Project:       query.Get("project"),
	}
	if query.Has("tags") {
		for _, tag := range query["tags"] {
			tag = strings.TrimSpace(tag)
			if len(tag) == 0 {
				continue
			}
			for _, sub := range strings.Split(tag, ",") {
				sub = strings.TrimSpace(sub)
				if len(sub) == 0 {
					continue
				}
				filter.Tags = append(filter.Tags, sub)
			}
		}
	}
	if query.Has("search_words") {
		for _, word := range query["search_words"] {
			word = strings.TrimSpace(word)
			if len(word) == 0 {
				continue
			}
			for _, sub := range strings.Split(word, " ") {
				sub = strings.TrimSpace(sub)
				if len(sub) == 0 {
					continue
				}
				filter.SearchWords = append(filter.SearchWords, sub)
			}
		}
	}

	return filter
}
