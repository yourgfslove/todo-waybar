# gtodo

Todo-приложение для Arch/Hyprland в виде единого Go-бинаря с тремя режимами работы:

- **TUI** — интерактивный интерфейс в floating-окне терминала (Bubbletea + Lipgloss)
- **Waybar** — JSON-вывод для модуля в статус-баре
- **CLI** — управление задачами из терминала и скриптов

Все режимы работают с одним хранилищем — JSON-файлом `~/.local/share/gtodo/tasks.json`,
поэтому задачи синхронны между баром, TUI и командной строкой.

---

## Возможности

- Приоритеты `high` / `mid` / `low` с цветными бейджами (🔴 🟡 🟢)
- Теги и опциональные дедлайны (подсветка просроченных и сегодняшних)
- Фильтры **All / Active / Done** и циклическая сортировка по приоритету → дедлайну → дате создания
- Фильтр по тегу
- Атомарная запись хранилища (через временный файл + `rename`)
- Поиск задачи по полному `id` или по уникальному префиксу (удобно в CLI)

---

## Установка

### Требования

- Go 1.26+
- Терминал (по умолчанию Alacritty) — для TUI-режима
- Waybar — для модуля в баре (опционально)

### Сборка

```bash
git clone <repo> gtodo && cd gtodo
go build -o gtodo .
cp gtodo ~/.local/bin/gtodo
```

Убедись, что `~/.local/bin` есть в `$PATH`:

```bash
command -v gtodo
```

---

## Использование

### Режим CLI

```bash
# Добавить задачу
gtodo add "Купить продукты" --priority high --tags home,personal --deadline 2026-06-01

# Список задач
gtodo list                      # все
gtodo list --filter active      # только активные
gtodo list --sort deadline      # сортировка по дедлайну
gtodo list --json               # вывод в JSON

# Отметить выполненной / снять отметку
gtodo done <id>
gtodo undone <id>

# Удалить
gtodo rm <id>

# Вывести JSON одной задачи
gtodo get <id>
```

Флаги `add`: `-p/--priority` (high/mid/low, по умолчанию `mid`),
`-t/--tags` (через запятую), `-d/--deadline` (формат `YYYY-MM-DD`).

`<id>` можно указывать сокращённо — достаточно уникального префикса
(например, первых 8 символов, которые показывает `gtodo list`).

### Режим TUI

```bash
gtodo tui
# или во floating-окне Alacritty:
alacritty --class gtodo -e gtodo tui
```

#### Клавиши

| Клавиша | Действие |
|---|---|
| `j` / `↓` | Вниз |
| `k` / `↑` | Вверх |
| `g` / `G` | В начало / в конец |
| `Space` | Toggle done/undone |
| `a` | Добавить задачу |
| `e` | Редактировать задачу |
| `t` | Редактировать теги |
| `d` | Удалить задачу |
| `p` | Сменить приоритет (high → mid → low) |
| `Tab` | Переключить фильтр (All / Active / Done) |
| `s` | Переключить сортировку |
| `/` | Фильтр по тегу (пустой ввод — сбросить) |
| `?` | Показать/скрыть помощь |
| `q` / `Esc` | Выйти |

В форме добавления/редактирования: `Tab`/`↑`/`↓` — переход между полями,
`Enter` — далее/сохранить, `←`/`→` на поле приоритета — смена значения, `Esc` — отмена.

### Режим Waybar

```bash
gtodo waybar
```

Выводит JSON для модуля Waybar:

```json
{
  "text": "✓ 3  5",
  "tooltip": "● Купить продукты [high] → сегодня\n● Позвонить врачу [mid] → 01 Jun",
  "class": "has-urgent"
}
```

- `text` — `✓ выполнено  всего`
- `tooltip` — активные задачи, отсортированные по приоритету (макс. 10)
- `class` — `has-urgent`, если есть просроченные или `high`-задачи

---

## Интеграция с Hyprland + Waybar

> Синтаксис правил окна ниже — для **Hyprland 0.53+** (`windowrule ... match:class`).
> На старых версиях использовался `windowrulev2 = float, class:gtodo`.

### Hyprland (`~/.config/hypr/hyprland.conf`)

```ini
windowrule = float on,      match:class ^gtodo$
windowrule = size 700 500,  match:class ^gtodo$
windowrule = center on,     match:class ^gtodo$
```

### Waybar (`~/.config/waybar/config.jsonc`)

```jsonc
"custom/gtodo": {
    "exec": "gtodo waybar",
    "interval": 30,
    "return-type": "json",
    "format": "{}",
    "tooltip": true,
    "on-click": "alacritty --class gtodo -e gtodo tui"
}
```

Добавь `"custom/gtodo"` в один из списков `modules-left/center/right`.
Waybar **не** перезагружается автоматически — после правок выполни
`omarchy-restart-waybar` (или перезапусти waybar вручную).

### Стили Waybar (`~/.config/waybar/style.css`)

```css
#custom-gtodo {
    color: @foreground;
    padding: 0 8px;
}

#custom-gtodo.has-urgent {
    color: #ff5555;
}
```

---

## Хранилище

**Путь:** `~/.local/share/gtodo/tasks.json`
(или `$XDG_DATA_HOME/gtodo/tasks.json`, если переменная задана).

Структура задачи:

```json
{
  "id": "uuid-v4",
  "text": "Купить продукты",
  "done": false,
  "priority": "high",
  "tags": ["home", "personal"],
  "deadline": "2026-06-01",
  "created_at": "2026-05-30T12:00:00Z"
}
```

| Поле | Тип | Описание |
|---|---|---|
| `id` | string (UUID v4) | Уникальный идентификатор |
| `text` | string | Текст задачи |
| `done` | bool | Статус выполнения |
| `priority` | string | `high` / `mid` / `low` |
| `tags` | []string | Произвольные теги |
| `deadline` | string (YYYY-MM-DD) | Опциональный дедлайн |
| `created_at` | string (RFC3339) | Дата создания |

---

## Структура проекта

```
gtodo/
├── cmd/
│   ├── root.go       # cobra root, общие хелперы
│   ├── tui.go        # gtodo tui
│   ├── waybar.go     # gtodo waybar
│   └── cli.go        # gtodo add / list / done / undone / rm / get
├── internal/
│   ├── model/
│   │   └── task.go   # Task, Priority, дедлайны
│   ├── store/
│   │   └── json.go   # чтение/запись tasks.json, поиск, сортировки
│   └── ui/
│       ├── list.go   # корневая модель Bubbletea
│       ├── form.go   # форма добавления/редактирования
│       └── styles.go # стили Lipgloss
├── go.mod
├── go.sum
└── main.go
```

## Зависимости

- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)
- [github.com/google/uuid](https://github.com/google/uuid)
- [github.com/spf13/cobra](https://github.com/spf13/cobra)
