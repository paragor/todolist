package models

import (
	"reflect"
	"strconv"
	"testing"
)

func TestTask_Unify_tags(t1 *testing.T) {
	tests := []struct {
		tags     []string
		expected []string
	}{
		{
			tags:     []string{},
			expected: []string{},
		},
		{
			tags:     []string{""},
			expected: []string{},
		},
		{
			tags:     []string{"a"},
			expected: []string{"a"},
		},
		{
			tags:     []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			tags:     []string{"a", "b", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
	}
	for i, tt := range tests {
		t1.Run(strconv.Itoa(i), func(t1 *testing.T) {
			t := &Task{
				Tags: tt.tags,
			}
			t.Unify()
			if !reflect.DeepEqual(t.Tags, tt.expected) {
				t1.Errorf("tags should be:\n%v\nhave:\n%v", tt.expected, t.Tags)
			}
		})
	}
}
