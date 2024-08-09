package httpserver

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/paragor/todo/pkg/httpserver/htmxtemplates"
	"github.com/paragor/todo/pkg/models"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

var templates *template.Template

func init() {
	functions := map[string]interface{}{
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
	}

	templates = template.New("").Funcs(functions)
	templates = must(templates.ParseFS(htmxtemplates.Components, "components/*.html"))
	templates = must(templates.ParseFS(htmxtemplates.Pages, "pages/*.html"))
}

func renderHtmx(template string, data any) (*bytes.Buffer, func(), error) {
	buffer := bytesBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	deferFn := func() {
		bytesBufferPool.Put(buffer)
	}
	if err := templates.ExecuteTemplate(buffer, template, data); err != nil {
		deferFn()
		return nil, func() {}, err
	}

	return buffer, deferFn, nil
}

func writeHtmx(writer http.ResponseWriter, template string, data any, status int) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	buffer, deferFn, err := renderHtmx(template, data)
	defer deferFn()
	if err != nil {
		http.Error(writer, "error on render", 500)
		return
	}
	writer.WriteHeader(status)
	_, _ = writer.Write(buffer.Bytes())
}

type filterContext struct {
	Enabled     bool
	Filter      *models.ListFilter
	AllProjects map[string]int
	AllTags     map[string]int
}

type listContext struct {
	Tasks         []*models.Task
	FilterContext filterContext
}
type groupedListComponentContext struct {
	ExpandAll     bool
	GroupedTasks  []models.TaskGroup
	FilterContext filterContext
}

func (c *listContext) groupByProjects() *groupedListComponentContext {
	return &groupedListComponentContext{
		FilterContext: c.FilterContext,
		GroupedTasks:  models.GroupTasksByProject(c.Tasks),
	}
}
func (c *listContext) agenda() *groupedListComponentContext {
	result := &groupedListComponentContext{
		FilterContext: c.FilterContext,
		GroupedTasks:  models.Agenda(c.Tasks),
	}
	result.ExpandAll = true
	result.FilterContext.Enabled = false
	return result
}

func (h *httpServer) htmxGenerateListContext(request *http.Request) (*listContext, error) {
	_ = request.ParseForm()
	tasks, err := h.repository.All()
	if err != nil {
		return nil, fmt.Errorf("cant list tasks: %w", err)
	}
	filter := queryToListFilter(request.Form)
	tasks = filter.Apply(tasks)
	uniqProjects := models.UniqProjects(tasks)
	uniqTags := models.UniqTags(tasks)
	models.SortTasks(tasks)
	return &listContext{
		Tasks: tasks,
		FilterContext: filterContext{
			Enabled:     true,
			Filter:      filter,
			AllProjects: uniqProjects,
			AllTags:     uniqTags,
		},
	}, nil
}
func (h *httpServer) htmxPageMain(writer http.ResponseWriter, request *http.Request) {
	context, err := h.htmxGenerateListContext(request)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}

	tasksHtml, deferFn, err := renderHtmx("component/list_tasks", context)
	defer deferFn()
	if err != nil {
		http.Error(writer, "error on render", 500)
		return
	}
	writeHtmx(writer, "page/index", template.HTML(tasksHtml.String()), 200)
}
func (h *httpServer) htmxPageTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}
	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}

	if task == nil {
		http.Error(writer, "task not found", 400)
		return
	}

	tasksHtml, deferFn, err := renderHtmx("component/task_card", task)
	defer deferFn()
	if err != nil {
		http.Error(writer, "error on render", 500)
		return
	}
	writeHtmx(writer, "page/index", template.HTML(tasksHtml.String()), 200)
}

func (h *httpServer) htmxPageProjects(writer http.ResponseWriter, request *http.Request) {
	context, err := h.htmxGenerateListContext(request)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}

	tasksHtml, deferFn, err := renderHtmx("component/list_tasks_by_groups", context.groupByProjects())
	defer deferFn()
	if err != nil {
		http.Error(writer, "error on render", 500)
		return
	}
	writeHtmx(writer, "page/index", template.HTML(tasksHtml.String()), 200)
}
func (h *httpServer) htmxPageAgenda(writer http.ResponseWriter, request *http.Request) {
	context, err := h.htmxGenerateListContext(request)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}

	tasksHtml, deferFn, err := renderHtmx("component/list_tasks_by_groups", context.agenda())
	defer deferFn()
	if err != nil {
		http.Error(writer, "error on render", 500)
		return
	}
	writeHtmx(writer, "page/index", template.HTML(tasksHtml.String()), 200)
}

func (h *httpServer) htmxGetTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}
	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}
	if task == nil {
		http.Error(writer, "task not found", 400)
		return
	}
	writer.Header().Set("HX-Reswap", "outerHTML")
	writeHtmx(writer, "component/task_card", task, 200)
}

type taskModalContext struct {
	Task           *models.Task
	ProjectOptions []string
	TagsOptions    []string
}

func (h *httpServer) htmxGenerateTaskModalContext(task *models.Task) (*taskModalContext, error) {
	filter := models.NewDefaultListFilter()
	filter.ShowCompleted = true
	tasks, err := h.repository.All()
	if err != nil {
		return nil, fmt.Errorf("cant list tasks: %w", err)
	}
	tasks = filter.Apply(tasks)

	projects := []string{}
	for project := range models.UniqProjects(tasks) {
		if project == models.ProjectSelectorEmpty {
			continue
		}
		projects = append(projects, project)
	}
	sort.Strings(projects)
	tags := []string{}
	for tag := range models.UniqTags(tasks) {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return &taskModalContext{
		Task:           task,
		ProjectOptions: projects,
		TagsOptions:    tags,
	}, nil
}
func (h *httpServer) htmxEditTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	timezone := request.Form.Get("timezone")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}

	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}
	if task == nil {
		http.Error(writer, "task not found", 400)
		return
	}
	if len(timezone) > 0 {
		tz, err := time.LoadLocation(timezone)
		if err != nil {
			http.Error(writer, "cant load timezone: "+err.Error(), 500)
			return
		}
		task.CreatedAt = task.CreatedAt.In(tz)
		if task.Due != nil {
			due := task.Due.In(tz)
			task.Due = &due
		}
		if task.Notify != nil {
			notify := task.Notify.In(tz)
			task.Notify = &notify
		}
	}
	context, err := h.htmxGenerateTaskModalContext(task)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}
	writeHtmx(writer, "component/task_modal", context, 200)
}

func (h *httpServer) htmxCopyTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	timezone := request.Form.Get("timezone")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}

	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}
	if task == nil {
		http.Error(writer, "task not found", 400)
		return
	}
	if len(timezone) > 0 {
		tz, err := time.LoadLocation(timezone)
		if err != nil {
			http.Error(writer, "cant load timezone: "+err.Error(), 500)
			return
		}
		task.CreatedAt = task.CreatedAt.In(tz)
		if task.Due != nil {
			due := task.Due.In(tz)
			task.Due = &due
		}
		if task.Notify != nil {
			notify := task.Notify.In(tz)
			task.Notify = &notify
		}
	}
	task = task.Clone(true)
	task.Description = ""
	context, err := h.htmxGenerateTaskModalContext(task)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}

	writeHtmx(writer, "component/task_modal", context, 200)
}

func (h *httpServer) htmxNewTask(writer http.ResponseWriter, request *http.Request) {
	task := models.NewTask()
	context, err := h.htmxGenerateTaskModalContext(task)
	if err != nil {
		http.Error(writer, "cant generate context: "+err.Error(), 500)
		return
	}

	writeHtmx(writer, "component/task_modal", context, 200)
}

func (h *httpServer) htmxSaveStatus(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	status := request.Form.Get("status")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	if len(status) == 0 {
		http.Error(writer, "status cant not be empty", 400)
		return
	}

	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}
	parsedStatus, err := models.NewTaskStatus(status)
	if err != nil {
		http.Error(writer, "invalid status", 400)
		return
	}
	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}

	if task == nil {
		if err != nil {
			http.Error(writer, "task not found", 400)
			return
		}
	}
	task.Status = parsedStatus
	if err := h.repository.Insert(task); err != nil {
		http.Error(writer, "cant save task: "+err.Error(), 500)
		return
	}

	writer.Header().Set("HX-Reswap", "outerHTML")
	writeHtmx(writer, "component/task_card", task, 200)
}

func (h *httpServer) htmxSaveTask(writer http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	UUID := request.Form.Get("uuid")
	status := request.Form.Get("status")
	description := request.Form.Get("description")
	timezone := request.Form.Get("timezone")
	due := request.Form.Get("due")
	notify := request.Form.Get("notify")
	if len(UUID) == 0 {
		http.Error(writer, "uuid cant not be empty", 400)
		return
	}
	if len(status) == 0 {
		http.Error(writer, "status cant not be empty", 400)
		return
	}
	if len(description) == 0 {
		http.Error(writer, "description cant not be empty", 400)
		return
	}
	if len(timezone) == 0 {
		http.Error(writer, "timezone cant not be empty", 400)
		return
	}

	parsedUUID, err := uuid.Parse(UUID)
	if err != nil {
		http.Error(writer, "cant parse UUID: %s"+err.Error(), 400)
		return
	}
	parsedStatus, err := models.NewTaskStatus(status)
	if err != nil {
		http.Error(writer, "invalid status", 400)
		return
	}
	isNewTask := false
	task, err := h.repository.Get(parsedUUID)
	if err != nil {
		http.Error(writer, "cant fetch task: "+err.Error(), 500)
		return
	}

	if task == nil {
		isNewTask = true
		task = models.NewTask()
		task.UUID = parsedUUID
	}
	task.Status = parsedStatus
	task.Description = description
	task.Project = strings.TrimSpace(strings.ToLower(request.Form.Get("project")))
	tags := []string{}
	for _, t := range strings.Split(request.Form.Get("tags"), ",") {
		t = strings.TrimSpace(t)
		if len(t) == 0 {
			continue
		}
		tags = append(tags, t)
	}
	task.Tags = tags
	dueTime, err := parseBrowserTime(due, timezone)
	if err != nil {
		http.Error(writer, "cant parse due: "+err.Error(), 400)
		return
	}
	task.Due = dueTime
	notifyTime, err := parseBrowserTime(notify, timezone)
	if err != nil {
		http.Error(writer, "cant parse notify: "+err.Error(), 400)
		return
	}
	task.Notify = notifyTime

	if err := h.repository.Insert(task); err != nil {
		http.Error(writer, "cant save task: "+err.Error(), 500)
		return
	}
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if isNewTask {
		writer.Header().Set("HX-Redirect", "/task?uuid="+task.UUID.String())
	} else {
		writer.Header().Set("HX-Refresh", "true")
	}
	writer.WriteHeader(200)
	_, _ = writer.Write([]byte("Success!"))
}

func parseBrowserTime(browserDatetime string, timezone string) (*time.Time, error) {
	if len(browserDatetime) == 0 {
		return nil, nil
	}
	zone, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("cant load timezone: %w", err)
	}
	result, err := time.ParseInLocation("2006-01-02T15:04", browserDatetime, zone)
	if err != nil {
		return nil, fmt.Errorf("cant parse timestamp: %w", err)
	}
	return &result, nil
}
