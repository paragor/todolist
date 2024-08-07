package httpserver

import (
	"github.com/paragor/todo/pkg/models"
	"net/url"
	"strings"
)

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
			if len(tag) > 0 {
				filter.Tags = append(filter.Tags, tag)
			}
		}
	}
	if query.Has("search_words") {
		for _, word := range query["search_words"] {
			word = strings.TrimSpace(word)
			if len(word) > 0 {
				filter.SearchWords = append(filter.SearchWords, word)
			}
		}
	}

	return filter
}
