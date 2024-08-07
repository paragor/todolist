package models

import (
	"time"
)

func truncateToDay(date time.Time) time.Time {
	date = date.In(time.Local)
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func Agenda(tasks []*Task) []TaskGroup {
	todayStart := truncateToDay(time.Now())
	todayTasks := []*Task{}
	for _, task := range tasks {
		if task.Due == nil {
			continue
		}
		due := truncateToDay(*task.Due)
		if !due.Equal(todayStart) {
			continue
		}
		todayTasks = append(todayTasks, task)
	}
	SortTasks(todayTasks)

	overdueTasks := []*Task{}
	for _, task := range tasks {
		if task.Due == nil {
			continue
		}
		due := truncateToDay(*task.Due)
		if due.After(todayStart) || due.Equal(todayStart) {
			continue
		}
		overdueTasks = append(overdueTasks, task)
	}
	SortTasks(overdueTasks)

	thisWeekTasks := []*Task{}
	for _, task := range tasks {
		if task.Due == nil {
			continue
		}
		due := truncateToDay(*task.Due)
		if due.Before(todayStart) || due.Equal(todayStart) {
			continue
		}
		if due.After(todayStart.Add(7 * 24 * time.Hour)) {
			continue
		}
		thisWeekTasks = append(thisWeekTasks, task)
	}
	SortTasks(thisWeekTasks)

	return []TaskGroup{
		{
			Group: "Today",
			Tasks: todayTasks,
		},
		{
			Group: "Next 7 days",
			Tasks: thisWeekTasks,
		},
		{
			Group: "Overdue",
			Tasks: overdueTasks,
		},
	}
}
