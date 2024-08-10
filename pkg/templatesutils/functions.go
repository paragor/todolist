package templatesutils

import (
	"strings"
	"time"
)

func GetFunctions() map[string]interface{} {
	return map[string]interface{}{
		"strings_contains": func(slice []string, item string) bool {
			for _, v := range slice {
				if v == item {
					return true
				}
			}
			return false
		},
		"join": strings.Join,
		"time_is_over": func(date *time.Time) bool {
			if date == nil {
				return false
			}
			return date.Before(time.Now())
		},
		"now": func() time.Time { return time.Now() },
	}
}
