# ssh-portfolio

Интерактивное портфолио в терминале, которое открывается по SSH:

```bash
ssh <ваш-домен>
```

Написано на Go с помощью [Wish](https://github.com/charmbracelet/wish) (SSH-сервер), [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI) и [Lip Gloss](https://github.com/charmbracelet/lipgloss) (стили). Один бинарник, никаких баз данных и внешних сервисов.

Форкай, меняй тексты на свои и выкладывай на свой сервер.

## Что внутри

- **Пиксельный баннер** с переливающимся градиентом. Цвет меняется клавишей `c`.
- **Вкладки**: о себе, навыки, контакты, статистика, змейка.
- **About**: текст «печатается» как на машинке, переносится по ширине терминала и прокручивается стрелками.
- **Навыки** с анимированными полосками.
- **Статистика**: номер посетителя (хранится в файле и переживает перезапуски), сколько человек онлайн, аптайм сервера.
- **Змейка** прямо в терминале.

## Быстрый старт

Нужен Go 1.21 или новее.

```bash
git clone https://github.com/<твой-ник>/<репозиторий>.git
cd <репозиторий>
go mod tidy
PORT=2223 go run .
```

В соседнем терминале:

```bash
ssh -p 2223 localhost
```

Переменные окружения:

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PORT`     | `22`         | порт, на котором слушает сервер |
| `HOST`     | пусто (все интерфейсы) | адрес для привязки |

При первом запуске в текущей папке создаются:

- `.ssh/id_ed25519` — ключ хоста сервера. Не публикуй его и не меняй без нужды, иначе у посетителей появится предупреждение о смене ключа.
- `visitors.txt` — счётчик посетителей.

## Как сделать под себя

Весь контент лежит в начале `main.go`, в блоке «Контент». Править нужно в таких местах.

| Что менять | Где |
|------------|-----|
| Строка под баннером | константа `subtitle` |
| Текст «обо мне» | константа `aboutText` |
| Навыки и уровни (0–100) | срез `skills` |
| Контакты | срез `contacts` (формат `"Название   значение"`) |
| Названия вкладок | срез `tabNames` |
| Слово в баннере | константа `word` и таблица `glyphs` |
| Цветовые палитры баннера | срез `palettes` |
| Скорость печати, кадров, змейки | константы `tickEvery`, `typeSpeed`, `snakeEvery` |
| Размер поля змейки | константы `gridW`, `gridH` |

### Свой текст в баннере

Каждая буква — сетка 5×5, где `#` это закрашенный пиксель, а `.` пустой. Чтобы написать другое слово, поменяй `word` и добавь недостающие буквы в `glyphs`. Например, буква `A`:

```go
'A': {".###.", "#...#", "#####", "#...#", "#...#"},
```

Баннер отрисован только буквами из `glyphs`. Если в `word` встретится буква, которой там нет, она будет пустой.

### Добавить или убрать вкладку

1. Добавь константу в блок `tabAbout, tabSkills, ...` (порядок констант = порядок вкладок).
2. Добавь название в `tabNames` на то же место.
3. Напиши функцию `viewИмя() string` по аналогии с `viewContacts`.
4. Добавь `case` в `switch m.tab` внутри `View()`.

Вкладка получает номер (`1`–`9`) и попадает в переключение по `Tab` и стрелкам автоматически.

### Управление

| Клавиши | Действие |
|---------|----------|
| `Tab` / `Shift+Tab`, `←` `→`, `h` `l`, `1`–`9` | переключение вкладок |
| `↑` `↓`, `j` `k` | прокрутка текста (на вкладке About) |
| `c` | сменить палитру баннера |
| `q`, `Ctrl+C` | выход |
| стрелки, `wasd`, `hjkl` | управление змейкой |
| `r`, `Enter`, пробел | перезапуск змейки |

## Деплой на VPS

Схема: портфолио занимает порт 22, поэтому системный `sshd` переезжает на другой порт (в примере `2222`, выбери свой), и для администрирования ты заходишь с флагом `-p`.

### 1. Освободить порт 22

Сначала открой новый порт в фаерволе (`ufw` и панель хостера, если она есть), потом:

```bash
sudo ufw allow 2222/tcp
sudo nano /etc/ssh/sshd_config      # Port 2222
sudo sshd -t                        # проверка конфига
sudo systemctl restart ssh
```

**Не закрывай текущую сессию**, пока в новом терминале не проверишь вход: `ssh -p 2222 user@your-host`.

### 2. Пользователь и сервис

```bash
sudo useradd --system --create-home --home-dir /var/lib/portfolio --shell /usr/sbin/nologin portfolio
sudo ufw allow 22/tcp
```

`/etc/systemd/system/portfolio.service`:

```ini
[Unit]
Description=SSH portfolio (Wish)
After=network.target

[Service]
User=portfolio
WorkingDirectory=/var/lib/portfolio
ExecStart=/usr/local/bin/portfolio
Restart=always
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

`AmbientCapabilities` позволяет непривилегированному пользователю слушать порт 22. Рабочая папка `/var/lib/portfolio` хранит ключ хоста и счётчик.

### 3. Сборка и запуск

Собирай локально и копируй бинарник на сервер (так не нужен Go на VPS):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o portfolio .   # для ARM: GOARCH=arm64
scp -P 2222 portfolio user@your-host:/tmp/
```

На сервере:

```bash
sudo mv /tmp/portfolio /usr/local/bin/portfolio
sudo systemctl daemon-reload
sudo systemctl enable --now portfolio
journalctl -u portfolio -f
```

### 4. Скрипт обновления

`~/.ssh/config` на твоём компьютере:

```
Host vps
    HostName your-host
    Port 2222
    User user
```

`deploy.sh`:

```bash
#!/usr/bin/env bash
set -e
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o portfolio .
scp portfolio vps:/tmp/portfolio
ssh -t vps 'sudo mv /tmp/portfolio /usr/local/bin/portfolio && sudo systemctl restart portfolio'
```

## Частые проблемы

**Предупреждение о смене ключа хоста.** Если раньше на этом адресе был обычный `sshd`, очисти старую запись: `ssh-keygen -R your-host`.

**`address already in use` в логах.** Порт 22 всё ещё занят системным `sshd`. Проверь `ss -tlnp | grep ':22 '` и убедись, что `sshd` переехал на другой порт.

**Нет цветов на сервере.** В `main()` цветовой профиль задан явно через `lipgloss.SetColorProfile(termenv.ANSI256)`. Без него под systemd (нет терминала) lipgloss отключает цвета. Если у тебя терминал с truecolor, можно поставить `termenv.TrueColor`.

**`go mod tidy` падает с ошибкой про `charm.land/ssh`.** Charm переводит пакеты на новые пути (`charm.land/...`), а проект написан под v1 на путях `github.com/charmbracelet/...`. Зафиксируй версии:

```bash
go get github.com/charmbracelet/wish@v1.4.7
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/lipgloss@v1.1.0
go get github.com/charmbracelet/log@v1.0.0
go get github.com/muesli/termenv
go mod tidy
```

Если `tidy` всё равно тянет новый `ssh`, закрепи его вручную: `go get github.com/charmbracelet/ssh@v0.0.0-20250128164007-98fd5ae11894`. Лучше закоммитить `go.mod` и `go.sum` после успешной сборки, тогда у остальных этой проблемы не будет.

**Терминал мелкий, текст обрезается.** Интерфейс подстраивается под размер окна, но баннер и вкладки занимают около 13 строк. На окнах ниже ~24 строк удобнее увеличить терминал.

## Безопасность

- Сервер принимает только интерактивные сессии (`activeterm`): выполнить команду через `ssh host command` нельзя.
- Доступа к оболочке нет, пользователь `portfolio` создан без shell и без права входа.
- Аутентификация не требуется, портфолио открыто для всех. Не добавляй в него ничего, что не должно быть публичным. В статистике посетителю показывается только его собственный IP.
- Не коммить `.ssh/` и `visitors.txt` (см. `.gitignore`).

## Структура

```
.
├── main.go        # весь код: контент, баннер, вкладки, змейка, сервер
├── go.mod
├── go.sum
└── README.md
```

