package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gtodo/internal/model"
	"gtodo/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeTagInput
)

type filter int

const (
	filterAll filter = iota
	filterActive
	filterDone
)

func (f filter) String() string {
	switch f {
	case filterActive:
		return "Active"
	case filterDone:
		return "Done"
	default:
		return "All"
	}
}

func (f filter) next() filter { return (f + 1) % 3 }

// Model — корневая модель TUI.
type Model struct {
	store *store.Store
	tasks []model.Task

	mode      mode
	filter    filter
	sort      store.SortMode
	tagFilter string

	cursor   int
	showHelp bool
	grouped  bool
	status   string

	form     form
	tagInput textinput.Model

	onChange func()

	width  int
	height int
}

// New создаёт модель из хранилища. onChange (может быть nil) вызывается
// после успешного сохранения — например, чтобы обновить модуль waybar.
func New(s *store.Store, onChange func()) (Model, error) {
	tasks, err := s.Load()
	if err != nil {
		return Model{}, err
	}
	ti := textinput.New()
	ti.Placeholder = "тег (пусто — сбросить)"
	ti.Prompt = "/"
	return Model{
		store:    s,
		tasks:    tasks,
		filter:   filterAll,
		sort:     store.SortPriority,
		grouped:  true,
		tagInput: ti,
		onChange: onChange,
	}, nil
}

func (m Model) Init() tea.Cmd { return nil }

// taskGroup — задачи одного главного тега.
type taskGroup struct {
	tag   string // "" — без тегов
	tasks []model.Task
}

// filtered возвращает задачи после фильтра и фильтра-по-тегу (без сортировки).
func (m Model) filtered() []model.Task {
	out := make([]model.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		switch m.filter {
		case filterActive:
			if t.Done {
				continue
			}
		case filterDone:
			if !t.Done {
				continue
			}
		}
		if m.tagFilter != "" && !hasTag(t, m.tagFilter) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// groupedVisible группирует видимые задачи по главному тегу.
// Группы — по алфавиту, «без тегов» в конце; внутри группы — по m.sort.
func (m Model) groupedVisible() []taskGroup {
	buckets := map[string][]model.Task{}
	for _, t := range m.filtered() {
		k := t.MainTag()
		buckets[k] = append(buckets[k], t)
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sortTagKeys(keys)

	groups := make([]taskGroup, 0, len(keys))
	for _, k := range keys {
		ts := buckets[k]
		store.Sort(ts, m.sort)
		groups = append(groups, taskGroup{tag: k, tasks: ts})
	}
	return groups
}

// visible возвращает видимые задачи в порядке отображения.
// Курсор индексирует именно этот плоский порядок.
func (m Model) visible() []model.Task {
	if !m.grouped {
		out := m.filtered()
		store.Sort(out, m.sort)
		return out
	}
	var out []model.Task
	for _, g := range m.groupedVisible() {
		out = append(out, g.tasks...)
	}
	return out
}

// sortTagKeys сортирует имена тегов по алфавиту, пустой ключ («без тегов») — в конец.
func sortTagKeys(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		if (keys[i] == "") != (keys[j] == "") {
			return keys[j] == ""
		}
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
}

func (m *Model) clampCursor(n int) {
	if n == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > n-1 {
		m.cursor = n - 1
	}
}

func (m *Model) persist() {
	if err := m.store.Save(m.tasks); err != nil {
		m.status = "ошибка сохранения: " + err.Error()
		return
	}
	if m.onChange != nil {
		m.onChange()
	}
}

// taskIndex возвращает индекс задачи в m.tasks по id.
func (m Model) taskIndex(id string) int {
	for i := range m.tasks {
		if m.tasks[i].ID == id {
			return i
		}
	}
	return -1
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeForm:
			return m.updateForm(msg)
		case modeTagInput:
			return m.updateTagInput(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vis := m.visible()
	m.status = ""

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
	case "j", "down":
		m.cursor++
		m.clampCursor(len(vis))
	case "k", "up":
		m.cursor--
		m.clampCursor(len(vis))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(vis) - 1
		m.clampCursor(len(vis))
	case " ":
		if cur, ok := current(vis, m.cursor); ok {
			i := m.taskIndex(cur.ID)
			m.tasks[i].Done = !m.tasks[i].Done
			m.persist()
		}
	case "p":
		if cur, ok := current(vis, m.cursor); ok {
			i := m.taskIndex(cur.ID)
			m.tasks[i].Priority = m.tasks[i].Priority.Next()
			m.persist()
		}
	case "d":
		if cur, ok := current(vis, m.cursor); ok {
			i := m.taskIndex(cur.ID)
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			m.persist()
			m.clampCursor(len(m.visible()))
		}
	case "a":
		m.mode = modeForm
		m.form = newForm(nil)
		return m, m.form.applyFocus()
	case "e":
		if cur, ok := current(vis, m.cursor); ok {
			m.mode = modeForm
			m.form = newForm(&cur)
			return m, m.form.applyFocus()
		}
	case "t":
		if cur, ok := current(vis, m.cursor); ok {
			m.mode = modeForm
			m.form = newForm(&cur)
			m.form.focus = fieldTags
			return m, m.form.applyFocus()
		}
	case "tab":
		m.filter = m.filter.next()
		m.clampCursor(len(m.visible()))
	case "s":
		m.sort = m.sort.Next()
		m.status = "сортировка: " + m.sort.String()
	case "f":
		m.grouped = !m.grouped
		m.clampCursor(len(m.visible()))
		if m.grouped {
			m.status = "режим: группы по тегам"
		} else {
			m.status = "режим: плоский список"
		}
	case "/":
		m.mode = modeTagInput
		m.tagInput.SetValue(m.tagFilter)
		return m, m.tagInput.Focus()
	}
	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f, action, cmd := m.form.update(msg)
	m.form = f
	switch action {
	case actionCancel:
		m.mode = modeList
		return m, nil
	case actionSubmit:
		return m.submitForm()
	}
	return m, cmd
}

func (m Model) submitForm() (tea.Model, tea.Cmd) {
	text, prio, tags, deadline := m.form.values()
	if text == "" {
		m.status = "текст задачи не может быть пустым"
		return m, m.form.applyFocus()
	}
	now := time.Now()
	stored, err := model.ParseDeadlineInput(deadline, now)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}

	if m.form.editingID == "" {
		m.tasks = append(m.tasks, model.Task{
			ID:        uuid.NewString(),
			Text:      text,
			Priority:  prio,
			Tags:      tags,
			Deadline:  stored,
			CreatedAt: now.UTC(),
		})
		m.status = "задача добавлена"
	} else if i := m.taskIndex(m.form.editingID); i >= 0 {
		m.tasks[i].Text = text
		m.tasks[i].Priority = prio
		m.tasks[i].Tags = tags
		m.tasks[i].Deadline = stored
		m.status = "задача обновлена"
	}
	m.persist()
	m.mode = modeList
	m.clampCursor(len(m.visible()))
	return m, nil
}

func (m Model) updateTagInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "enter":
		m.tagFilter = strings.TrimSpace(m.tagInput.Value())
		m.mode = modeList
		m.cursor = 0
		if m.tagFilter == "" {
			m.status = "фильтр по тегу сброшен"
		} else {
			m.status = "фильтр по тегу: " + m.tagFilter
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.mode == modeForm {
		return m.form.view() + "\n"
	}

	var b strings.Builder
	b.WriteString(m.headerView() + "\n\n")
	b.WriteString(m.listView())
	b.WriteString("\n")

	if m.mode == modeTagInput {
		b.WriteString(statusStyle.Render("Фильтр по тегу: ") + m.tagInput.View() + "\n")
	} else if m.status != "" {
		b.WriteString(statusStyle.Render(m.status) + "\n")
	}

	b.WriteString(m.footerView())
	return b.String()
}

func (m Model) headerView() string {
	filterPart := filterStyle.Render(m.filter.String())
	sortPart := "sort:" + m.sort.String()
	viewPart := "группы"
	if !m.grouped {
		viewPart = "плоский"
	}
	header := fmt.Sprintf("%s   фильтр: %s   %s   %s",
		titleStyle.Render(" gtodo "), filterPart, headerStyle.Render(sortPart), headerStyle.Render(viewPart))
	if m.tagFilter != "" {
		header += headerStyle.Render("#" + m.tagFilter)
	}
	return header
}

func (m Model) listView() string {
	if !m.grouped {
		return m.flatListView()
	}

	groups := m.groupedVisible()
	total := 0
	for _, g := range groups {
		total += len(g.tasks)
	}
	if total == 0 {
		return normalLineStyle.Render("  — нет задач —")
	}

	var b strings.Builder
	now := time.Now()
	idx := 0
	for _, g := range groups {
		b.WriteString(groupHeaderView(g.tag, len(g.tasks)) + "\n")
		for _, t := range g.tasks {
			b.WriteString(m.taskLine(t, idx, now) + "\n")
			idx++
		}
	}
	return b.String()
}

func (m Model) flatListView() string {
	vis := m.visible()
	if len(vis) == 0 {
		return normalLineStyle.Render("  — нет задач —")
	}
	var b strings.Builder
	now := time.Now()
	for i, t := range vis {
		b.WriteString(m.taskLine(t, i, now) + "\n")
	}
	return b.String()
}

func groupHeaderView(tag string, n int) string {
	name := tag
	if name == "" {
		name = "без тегов"
	}
	return groupHeaderStyle.Render(name) + " " + groupCountStyle.Render(fmt.Sprintf("(%d)", n))
}

// taskLine рендерит одну задачу с отступом под заголовком группы. i — её
// позиция в плоском порядке (для курсора).
func (m Model) taskLine(t model.Task, i int, now time.Time) string {
	// В режиме групп задачи с отступом под заголовком; в плоском — без отступа.
	marker := "  "
	cursorMark := cursorStyle.Render("▌ ")
	if m.grouped {
		marker = "     "
		cursorMark = "   " + cursorStyle.Render("▌ ")
	}
	if i == m.cursor {
		marker = cursorMark
	}

	box := "[ ]"
	if t.Done {
		box = "[x]"
	}

	lineStyle := normalLineStyle
	if t.Done {
		lineStyle = doneLineStyle
	} else if i == m.cursor {
		lineStyle = selectedLineStyle
	}

	// В группах главный тег — это заголовок, поэтому в строке только подтеги.
	// В плоском режиме показываем все теги, чтобы не терять контекст.
	tagLabel := t.SubTagsLabel()
	if !m.grouped {
		tagLabel = t.AllTagsLabel()
	}

	seg := box + " " + priorityBadge(t.Priority) + " "
	if tagLabel != "" {
		seg += tagStyle.Render(tagLabel) + " "
	}
	seg += lineStyle.Render(t.Text)

	out := marker + seg
	if label := t.DeadlineLabel(now); label != "" {
		out += " " + deadlineView(t, label)
	}
	return out
}

func deadlineView(t model.Task, label string) string {
	st := lipgloss.NewStyle().Foreground(colSubtle)
	switch {
	case t.Overdue():
		st = lipgloss.NewStyle().Foreground(colOverdue).Bold(true)
	case t.DueToday():
		st = lipgloss.NewStyle().Foreground(colToday).Bold(true)
	}
	return st.Render("→ " + label)
}

func (m Model) footerView() string {
	if m.showHelp {
		return helpStyle.Render(fullHelp)
	}
	return helpStyle.Render("j/k — навигация • space — done • a — добавить • e — правка • d — удалить • f — группы/плоский • ? — помощь • q — выход")
}

const fullHelp = "Навигация:  j/↓ вниз   k/↑ вверх   g/G начало/конец\n" +
	"Действия:   space toggle done   a добавить   e правка   t теги   d удалить   p приоритет\n" +
	"Вид:        Tab фильтр (All/Active/Done)   s сортировка   f группы/плоский   / фильтр по тегу\n" +
	"Прочее:     ? помощь   q/Esc выход"

func current(vis []model.Task, cursor int) (model.Task, bool) {
	if cursor < 0 || cursor >= len(vis) {
		return model.Task{}, false
	}
	return vis[cursor], true
}

func hasTag(t model.Task, tag string) bool {
	for _, x := range t.Tags {
		if strings.EqualFold(x, tag) {
			return true
		}
	}
	return false
}
