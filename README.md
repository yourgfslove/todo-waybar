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
- Теги и опциональные дедлайны — абсолютной датой/временем **или** относительным сроком от момента создания (`--in 2d3h30m`)
- Подсветка просроченных и сегодняшних, живой отсчёт остатка времени
- Группировка по тегам: **главный тег** (первый в списке) задаёт группу, остальные — подтеги-метки `#sub`; работает в TUI и в tooltip Waybar
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
# Добавить задачу с абсолютным дедлайном (дата или дата+время)
gtodo add "Купить продукты" --priority high --tags home,personal --deadline 2026-06-01
gtodo add "Встреча" --deadline "2026-06-01 14:30"

# …или с относительным сроком от момента создания
gtodo add "Сдать отчёт" --priority high --in 2d3h30m
gtodo add "Позвонить" --in 45m

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

Флаги `add`:
- `-p/--priority` — high/mid/low (по умолчанию `mid`)
- `-t/--tags` — теги через запятую
- `-d/--deadline` — абсолютный дедлайн: `YYYY-MM-DD` или `"YYYY-MM-DD HH:MM"`
- `-i/--in` — относительный срок от создания: `2d3h30m`, `45m`, `1w` (единицы: `w` недели, `d` дни, `h` часы, `m` минуты; можно комбинировать)

`--deadline` и `--in` взаимоисключающие. `<id>` можно указывать сокращённо —
достаточно уникального префикса (первых 8 символов из `gtodo list`).

**Группировка по тегам.** Задачи делятся на группы по **главному тегу** — это
**первый** тег в списке (`--tags work,urgent,backend` → группа `work`). Остальные
теги — подтеги, показываются как метки `#urgent #backend` перед текстом задачи.
Задачи без тегов попадают в группу «без тегов». Группы идут по алфавиту
(«без тегов» — в конце), внутри группы действует текущая сортировка.

В tooltip Waybar группировка включена всегда. В TUI её можно переключать
клавишей `f`: в плоском режиме заголовков групп нет, задачи отсортированы
единым списком, а перед текстом показываются **все** теги (`#work #urgent`) —
удобно, когда групп много. Текущий режим виден в шапке (`группы` / `плоский`).

**Как хранится и отображается дедлайн:**

- `--in` фиксируется в момент создания (`created_at + срок`) и хранится как точное время.
- Срок со временем показывается как остаток + сам момент: `→ через 2ч 30м (30 May 18:45)`,
  просроченный — `→ просрочено на 1ч (30 May 16:00)`. Остаток пересчитывается «вживую».
- Абсолютная дата без времени отображается как раньше: `→ 01 Jun`, `→ сегодня`, `→ просрочено (01 May)`.

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
| `f` | Переключить вид: группы по тегам ⇄ плоский список |
| `/` | Фильтр по тегу (пустой ввод — сбросить) |
| `?` | Показать/скрыть помощь |
| `q` / `Esc` | Выйти |

В форме добавления/редактирования: `Tab`/`↑`/`↓` — переход между полями,
`Enter` — далее/сохранить, `←`/`→` на поле приоритета — смена значения, `Esc` — отмена.
Поле «Дедлайн» принимает и относительный срок (`2d3h30m`), и дату (`2026-06-01`),
и дату со временем (`2026-06-01 14:30`).

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
    "signal": 10,
    "return-type": "json",
    "format": "{}",
    "tooltip": true,
    "on-click": "alacritty --class gtodo -e gtodo tui"
}
```

Добавь `"custom/gtodo"` в один из списков `modules-left/center/right`.
Waybar **не** перезагружается автоматически — после правок выполни
`omarchy-restart-waybar` (или перезапусти waybar вручную).

**Мгновенное обновление.** Помимо опроса раз в `interval`, модуль обновляется
по сигналу `SIGRTMIN+10` (`"signal": 10`). После любой мутации задач —
из CLI (`add`/`done`/`undone`/`rm`) и из TUI — `gtodo` сам шлёт этот сигнал
(`pkill -RTMIN+10 waybar`), так что бар обновляется сразу, не дожидаясь 30 с.
Номер сигнала задан константой `waybarSignal` в `cmd/waybar.go` — если поменяешь
его в конфиге, поменяй и там.

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
