package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

// ───────────────────────── Контент (правь под себя) ─────────────────────────

const subtitle = "romanbelx · devops & fullstack  · curious   —   z.belx.tech"

const aboutText = `О, ты здесь! Ну тогда я расскажу о себе, если ты не против).

Моей основной деятельностью/увлечением является фуллстек-разработка, тестирование и девопс.

Я люблю react и bun js, меня поражает и вдохновляет гошка и возможность создавать именные проекты с нуля
до уровня, когда ты можешь увидеть и извлечь какую-либо выгоду)


Технологии, которые я знаю и с которыми работаю:
React, Next.js, tailwind css - мой основной вектор, но я работал и со многими другими в вебе: vue, htmx и т.д.

Bun, nest, express, golang и django - фреймворки, на которых я писал или пишу проекты, особенно я люблю bun, а вы?

Sqlite, postgres, mysql - базы, которые я знаю и уважаю, но больше всего я люблю sqlite и psql
sqlite за простоту и скорость, а postgres за многофункциональность

Redis и memcache - кеширование, которое уже делал

В девопсе я тоже не отстал, конечно:
- Доверяю только debian 12 и 13
- Юзаю docker, как без него
- Про ssh, fail2ban, ufw, iptables и т.п. я даже не говорю
- Начинаю освоение kubernetes - к сожалению, у меня не было с ним опыта)
- Системы логирования я трогал через netdata
К сожалению, не приходилось использовать prometheus + grafana, но, возможно, когда вы это читаете, я уже развернул

Про ci/cd: это всеми любимый, наверное, github actions. Jenkins и gitlab я не трогал, так как проекты не такие масштабные

Ansible и terraform я в курсе, ребят

Также я знаю английский язык на уровне b2-c1 - много лет занимался дополнительно

Про алгоритмическую базу не забываю: решаю leetcode, сейчас у меня 82 решенные задачи
А если точнее, то из листа blind75 решил уже 60 задач easy и medium

Учился в академии Калашников 4 года в направлении it, успел даже попреподавать 1 месяц)

Не токсичу, принимаю людей такими, какие они есть, и пытаюсь найти солюшен в любой ситуации

Не хочу себя восхвалять, но я всегда работаю по принципу поглощения информации и опыта

Сложно и ново -> я уже тут пытаюсь это реализовать :)

Про проекты я расскажу лично вам, если это вам интересно -> не забывай про контакты)

Я учусь в ИПЭК на тестировщика и очень хочу найти работу, если я вас заинтересовал, пожалуйста, свяжитесь со мной

Ну и напоследок загляни во вкладку Змейка :)`

type skill struct {
	name  string
	level int
}

var skills = []skill{
	{"TypeScript", 90},
	{"Bun / Node", 85},
	{"Linux", 80},
	{"Go", 50},
	{"Docker", 65},
	{"Git", 70},
	{"Ansible", 30},
	{"Terraform", 30},
	{"Kubernetes", 5},
	{"CI/CD(github actions)", 70},
	{"Postgres,Redis,Sqlite", 50},
	{"Английский язык(c1)", 90},
}

var contacts = []string{
	"GitHub    github.com/Belxz777",
	"Telegram  @belyxz",
	"Email     x@belx.tech",
}

// ───────────────────────── Пиксельный баннер ─────────────────────────

const word = "BELX"

var glyphs = map[rune][5]string{
	'B': {"####.", "#...#", "####.", "#...#", "####."},
	'E': {"#####", "#....", "####.", "#....", "#####"},
	'L': {"#....", "#....", "#....", "#....", "#####"},
	'X': {"#...#", ".#.#.", "..#..", ".#.#.", "#...#"},
}

// палитры: начальный оттенок и размах (в градусах HSV)
var palettes = [][2]float64{{0, 360}, {170, 140}, {90, 80}, {0, 60}}

func hsvHex(h, s, v float64) string {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return fmt.Sprintf("#%02x%02x%02x", int((r+m)*255), int((g+m)*255), int((b+m)*255))
}

func pixelColor(x, y, frame, pal int) lipgloss.Color {
	p := palettes[pal]
	t := 0.5 + 0.5*math.Sin(float64(x)*0.28+float64(y)*0.2-float64(frame)*0.06)
	return lipgloss.Color(hsvHex(p[0]+p[1]*t, 0.75, 1))
}

func renderBanner(frame, pal int, wide bool) string {
	px, gap := "██", "  "
	if !wide {
		px, gap = "█", " "
	}
	empty := strings.Repeat(" ", lipgloss.Width(px))

	var sb strings.Builder
	for y := 0; y < 5; y++ {
		col := 0
		for li, ch := range word {
			g := glyphs[ch]
			for x := 0; x < 5; x++ {
				if g[y][x] == '#' {
					st := lipgloss.NewStyle().Foreground(pixelColor(col+x, y, frame, pal))
					sb.WriteString(st.Render(px))
				} else {
					sb.WriteString(empty)
				}
			}
			col += 6
			if li < len(word)-1 {
				sb.WriteString(gap)
			}
		}
		if y < 4 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ───────────────────────── Глобальная статистика ─────────────────────────

const counterFile = "visitors.txt"

var (
	startedAt = time.Now()
	online    int32
	visitors  int64
	counterMu sync.Mutex
)

func loadCounter() {
	b, err := os.ReadFile(counterFile)
	if err != nil {
		return
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	visitors = n
}

func nextVisitor() int64 {
	counterMu.Lock()
	defer counterMu.Unlock()
	visitors++
	_ = os.WriteFile(counterFile, []byte(strconv.FormatInt(visitors, 10)), 0o644)
	return visitors
}

// ───────────────────────── Стили ─────────────────────────

var (
	dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accent   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	tabOn    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Underline(true)
	linkSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Underline(true)
	foodSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f87"))
	headSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b8ff6a"))
	bodySt   = lipgloss.NewStyle().Foreground(lipgloss.Color("#5faf5f"))
	boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

// ───────────────────────── Модель ─────────────────────────

const (
	tabAbout = iota
	tabSkills
	tabContacts
	tabSnake
	tabStats
)

var tabNames = []string{"Хто я?", "Что могешь?", "Пиши сюда", "Змейка", "Инфо для вас"}

const (
	gridW = 24
	gridH = 12

	tickEvery  = 40 * time.Millisecond // частота обновления экрана (25 кадров/с)
	typeSpeed  = 4                     // символов "печатается" за кадр (~300 симв/с)
	snakeEvery = 3                     // змейка делает шаг раз в N кадров (~120 мс)
	typeDone   = 1 << 20
)

type point struct{ x, y int }

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(tickEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type model struct {
	width, height int
	tab           int
	frame         int
	pal           int
	typed         int
	skillAnim     int
	scroll        int // на сколько строк "About" прокручен вверх от низа

	aboutW     int
	aboutCache []string

	addr    string
	user    string
	visitor int64
	since   time.Time

	snake   []point
	dir     point
	queue   []point // очередь поворотов, чтобы быстрые нажатия не терялись
	food    point
	score   int
	best    int
	dead    bool
	started bool
}

func (m *model) Init() tea.Cmd { return tick() }

func (m *model) setTab(i int) {
	m.tab = (i + len(tabNames)) % len(tabNames)
	switch m.tab {
	case tabAbout:
		m.typed = 0
		m.scroll = 0
	case tabSkills:
		m.skillAnim = 0
	case tabSnake:
		m.resetSnake()
	}
}

// ── About: перенос строк и прокрутка ──

func (m *model) aboutLines() []string {
	w := 80
	if m.width > 8 {
		w = m.width - 4
	}
	if m.aboutCache == nil || m.aboutW != w {
		wrapped := lipgloss.NewStyle().Width(w).Render(aboutText)
		lines := strings.Split(wrapped, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
		m.aboutCache, m.aboutW = lines, w
	}
	return m.aboutCache
}

// сколько строк текста помещается на экране (0 — размер терминала неизвестен)
func (m *model) aboutAvail() int {
	if m.height == 0 {
		return 0
	}
	return max(5, m.height-14)
}

func (m *model) maxScroll() int {
	avail := m.aboutAvail()
	if avail == 0 {
		return 0
	}
	return max(0, len(m.aboutLines())-avail)
}

// ── змейка ──

func (m *model) onSnake(p point) bool {
	for _, s := range m.snake {
		if s == p {
			return true
		}
	}
	return false
}

func (m *model) placeFood() {
	if len(m.snake) >= gridW*gridH {
		return
	}
	for {
		p := point{rand.Intn(gridW), rand.Intn(gridH)}
		if !m.onSnake(p) {
			m.food = p
			return
		}
	}
}

func (m *model) resetSnake() {
	m.snake = []point{{5, 6}, {4, 6}, {3, 6}}
	m.dir = point{1, 0}
	m.queue = nil
	m.score = 0
	m.dead = false
	m.started = false
	m.placeFood()
}

func (m *model) steer(d point) {
	if m.dead {
		return
	}
	m.started = true
	last := m.dir
	if n := len(m.queue); n > 0 {
		last = m.queue[n-1]
	}
	if d == last || (d.x == -last.x && d.y == -last.y) {
		return
	}
	if len(m.queue) < 2 {
		m.queue = append(m.queue, d)
	}
}

func (m *model) stepSnake() {
	if len(m.queue) > 0 {
		m.dir = m.queue[0]
		m.queue = m.queue[1:]
	}
	head := m.snake[0]
	nh := point{head.x + m.dir.x, head.y + m.dir.y}
	if nh.x < 0 || nh.y < 0 || nh.x >= gridW || nh.y >= gridH || m.onSnake(nh) {
		m.dead = true
		if m.score > m.best {
			m.best = m.score
		}
		return
	}
	m.snake = append([]point{nh}, m.snake...)
	if nh == m.food {
		m.score++
		m.placeFood()
	} else {
		m.snake = m.snake[:len(m.snake)-1]
	}
}

// ── Update ──

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.aboutCache = nil
		m.scroll = min(m.scroll, m.maxScroll())

	case tickMsg:
		m.frame++
		if m.typed < typeDone {
			m.typed += typeSpeed
		}
		if m.skillAnim < 100 {
			m.skillAnim += 3
		}
		if m.tab == tabSnake && m.started && !m.dead && m.frame%snakeEvery == 0 {
			m.stepSnake()
		}
		return m, tick()

	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.setTab(m.tab + 1)
			return m, nil
		case "shift+tab":
			m.setTab(m.tab - 1)
			return m, nil
		case "c":
			m.pal = (m.pal + 1) % len(palettes)
			return m, nil
		}
		if len(k) == 1 && k[0] >= '1' && int(k[0]-'1') < len(tabNames) {
			m.setTab(int(k[0] - '1'))
			return m, nil
		}

		if m.tab == tabSnake {
			switch k {
			case "up", "w", "k":
				m.steer(point{0, -1})
			case "down", "s", "j":
				m.steer(point{0, 1})
			case "left", "a", "h":
				m.steer(point{-1, 0})
			case "right", "d", "l":
				m.steer(point{1, 0})
			case "r", "enter", " ":
				m.resetSnake()
			}
			return m, nil
		}

		switch k {
		case "right", "l":
			m.setTab(m.tab + 1)
		case "left", "h":
			m.setTab(m.tab - 1)
		case "up", "k":
			if m.tab == tabAbout {
				m.typed = typeDone // прокрутка пропускает анимацию печати
				m.scroll = min(m.scroll+1, m.maxScroll())
			}
		case "down", "j":
			if m.tab == tabAbout {
				m.typed = typeDone
				m.scroll = max(m.scroll-1, 0)
			}
		}
	}
	return m, nil
}

// ── View ──

func (m *model) renderTabs() string {
	var parts []string
	for i, n := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, n)
		if i == m.tab {
			parts = append(parts, tabOn.Render(label))
		} else {
			parts = append(parts, dim.Render(label))
		}
	}
	return strings.Join(parts, "   ")
}

func (m *model) viewAbout() string {
	full := []rune(strings.Join(m.aboutLines(), "\n"))
	n := min(m.typed, len(full))
	typing := n < len(full)

	shown := strings.Split(string(full[:n]), "\n")
	if typing || (m.frame/12)%2 == 0 {
		shown[len(shown)-1] += "▌"
	}

	if avail := m.aboutAvail(); avail > 0 && len(shown) > avail {
		off := 0
		if !typing {
			off = min(m.scroll, len(shown)-avail)
		}
		end := len(shown) - off
		shown = shown[end-avail : end]
	}
	return strings.Join(shown, "\n")
}

func (m *model) viewSkills() string {
	const barW = 24
	var b strings.Builder
	for i, s := range skills {
		cur := min(s.level, m.skillAnim)
		filled := cur * barW / 100
		bar := lipgloss.NewStyle().Foreground(pixelColor(i*4, 0, m.frame, m.pal)).
			Render(strings.Repeat("█", filled))
		bar += dim.Render(strings.Repeat("░", barW-filled))
		b.WriteString(fmt.Sprintf("%-22s %s %3d%%\n", s.name, bar, cur))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) viewStats() string {
	rows := [][2]string{
		{"ты — посетитель", fmt.Sprintf("#%d", m.visitor)},
		{"сейчас онлайн", strconv.Itoa(int(atomic.LoadInt32(&online)))},
		{"твой адрес", m.addr},
		{"пользователь", m.user},
		{"в сессии", time.Since(m.since).Round(time.Second).String()},
		{"аптайм сервера", time.Since(startedAt).Round(time.Second).String()},
		{"время сервера", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")},
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("%s %s\n", dim.Render(fmt.Sprintf("%-16s", r[0])), r[1]))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) viewSnake() string {
	var grid strings.Builder
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			p := point{x, y}
			switch {
			case p == m.snake[0]:
				grid.WriteString(headSt.Render("██"))
			case m.onSnake(p):
				grid.WriteString(bodySt.Render("▓▓"))
			case p == m.food:
				grid.WriteString(foodSt.Render("◆ "))
			default:
				grid.WriteString(dim.Render("· "))
			}
		}
		if y < gridH-1 {
			grid.WriteString("\n")
		}
	}
	status := fmt.Sprintf("score %d   best %d", m.score, m.best)
	switch {
	case m.dead:
		status = accent.Render("потеряно") + dim.Render(" — r чтобы заново") + "   " + status
	case !m.started:
		status = dim.Render("нажми стрелку, чтобы начать") + "   " + status
	}
	return boxStyle.Render(grid.String()) + "\n" + status
}

func (m *model) viewContacts() string {
	var b strings.Builder
	for _, c := range contacts {
		k, v, _ := strings.Cut(c, " ")
		v = strings.TrimSpace(v)
		b.WriteString(fmt.Sprintf("%s %s\n", dim.Render(fmt.Sprintf("%-9s", k)), linkSt.Render(v)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) View() string {
	var b strings.Builder

	compact := m.tab == tabSnake && m.height > 0 && m.height < 30
	if !compact {
		wide := m.width == 0 || m.width >= 56
		b.WriteString(renderBanner(m.frame, m.pal, wide))
		b.WriteString("\n" + dim.Render(subtitle) + "\n\n")
	}

	b.WriteString(m.renderTabs() + "\n\n")

	switch m.tab {
	case tabAbout:
		b.WriteString(m.viewAbout())
	case tabSkills:
		b.WriteString(m.viewSkills())
	case tabStats:
		b.WriteString(m.viewStats())
	case tabSnake:
		b.WriteString(m.viewSnake())
	case tabContacts:
		b.WriteString(m.viewContacts())
	}

	help := "tab / ←→ листать • c цвет • q выход"
	switch m.tab {
	case tabAbout:
		help = "tab / ←→ листать • ↑↓ прокрутка • c цвет • q выход"
	case tabSnake:
		help = "стрелки / wasd — управление • r заново • tab / ←→ листать • q выход"
	}
	b.WriteString("\n\n" + dim.Render(help))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// ───────────────────────── Сервер ─────────────────────────

func handler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	atomic.AddInt32(&online, 1)
	go func() {
		<-s.Context().Done()
		atomic.AddInt32(&online, -1)
	}()

	host, _, err := net.SplitHostPort(s.RemoteAddr().String())
	if err != nil {
		host = s.RemoteAddr().String()
	}

	m := &model{
		addr:    host,
		user:    s.User(),
		visitor: nextVisitor(),
		since:   time.Now(),
	}
	if pty, _, ok := s.Pty(); ok {
		m.width, m.height = pty.Window.Width, pty.Window.Height
	}
	m.resetSnake()
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func main() {
	// Под systemd у процесса нет TTY, и lipgloss отключает цвета.
	// Задаём профиль явно, чтобы цвета были и на VPS.
	lipgloss.SetColorProfile(termenv.ANSI256)
	loadCounter()

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "22"
	}

	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal("server", "err", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("listening", "addr", srv.Addr)
	go func() {
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("serve", "err", err)
			done <- nil
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
