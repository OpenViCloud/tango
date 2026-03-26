package onboarding

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type option struct {
	label string
	value string
	hint  string
}

type step int

const (
	stepDBDriver step = iota
	stepDatabaseURL
	stepChannel
	stepModel
	stepAPIKey
	stepReview
	stepSaved
)

type Model struct {
	step         step
	dbIndex      int
	channelIndex int
	input        textinput.Model
	config       Config
	width        int
	height       int
	err          string
	savePath     string
}

var dbOptions = []option{
	{
		label: "SQLite",
		value: "sqlite",
		hint:  "Lightweight local file DB. Great for onboarding and demos.",
	},
	{
		label: "PostgreSQL",
		value: "postgres",
		hint:  "Production-friendly relational DB with stronger concurrency.",
	},
}

var channelOptions = []option{
	{
		label: "Discord",
		value: "discord",
		hint:  "Bot-first chat workflow for teams and internal assistants.",
	},
	{
		label: "Telegram",
		value: "telegram",
		hint:  "Useful for mobile-first chat and lightweight bot integrations.",
	},
	{
		label: "Slack",
		value: "slack",
		hint:  "Good when the workspace already runs on Slack.",
	},
	{
		label: "Web Chat",
		value: "web",
		hint:  "Embed the experience directly in your web application.",
	},
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("31")).
			Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("67")).
			Padding(1, 2).
			MarginTop(1)

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("159"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("123"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("109"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("210")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("110"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("122")).
			Bold(true)
)

func NewModel() Model {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 256
	input.Width = 56

	return Model{
		step:         stepDBDriver,
		dbIndex:      0,
		channelIndex: 0,
		input:        input,
		config: Config{
			DBDriver:    dbOptions[0].value,
			DatabaseURL: defaultDatabaseURL(dbOptions[0].value),
			ChatChannel: channelOptions[0].value,
			ChatModel:   "gpt-4.1-mini",
		},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = min(64, max(36, msg.Width-16))
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step == stepSaved {
				return m, tea.Quit
			}
			if m.step > stepDBDriver {
				m = m.goBack()
				return m, nil
			}
			return m, tea.Quit
		}

		switch m.step {
		case stepDBDriver:
			return m.updateDBStep(msg)
		case stepChannel:
			return m.updateChannelStep(msg)
		case stepDatabaseURL, stepModel, stepAPIKey:
			return m.updateInputStep(msg)
		case stepReview:
			return m.updateReviewStep(msg)
		case stepSaved:
			if msg.String() == "enter" || msg.String() == "q" {
				return m, tea.Quit
			}
		}
	}

	if m.step == stepDatabaseURL || m.step == stepModel || m.step == stepAPIKey {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	header := titleStyle.Render("Tango Onboarding")
	subtitle := mutedStyle.Render("Configure a starter stack for local development or a first deployment.")

	body := ""
	switch m.step {
	case stepDBDriver:
		body = m.renderOptionStep(
			"Choose a database",
			"Start with the storage engine you want the app to boot against.",
			dbOptions,
			m.dbIndex,
		)
	case stepDatabaseURL:
		body = m.renderInputStep(
			"Database connection",
			"Edit the DSN if needed. The default is prefilled based on your database choice.",
		)
	case stepChannel:
		body = m.renderOptionStep(
			"Choose a chat channel",
			"Pick the first channel you want to wire into the app.",
			channelOptions,
			m.channelIndex,
		)
	case stepModel:
		body = m.renderInputStep(
			"Choose a chat model",
			"Examples: gpt-4.1-mini, claude-3-5-sonnet, local-llama3.1.",
		)
	case stepAPIKey:
		body = m.renderInputStep(
			"Paste an API key",
			"The wizard saves it into the app config file. If CONFIG_FILE_ENCRYPTION_KEY is set, sensitive fields are encrypted.",
		)
	case stepReview:
		body = m.renderReview()
	case stepSaved:
		body = m.renderSaved()
	}

	help := helpStyle.Render("Up/Down: move  Enter: confirm  Esc: back  Ctrl+C: quit")
	return lipgloss.JoinVertical(lipgloss.Left, header, subtitle, body, help)
}

func (m Model) updateDBStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left", "k", "shift+tab":
		m.dbIndex = (m.dbIndex - 1 + len(dbOptions)) % len(dbOptions)
		m.err = ""
		return m, nil
	case "down", "right", "j", "tab":
		m.dbIndex = (m.dbIndex + 1) % len(dbOptions)
		m.err = ""
		return m, nil
	case "enter":
		selected := dbOptions[m.dbIndex]
		m.config.DBDriver = selected.value
		m.config.DatabaseURL = defaultDatabaseURL(selected.value)
		m.step = stepDatabaseURL
		m.err = ""
		m.prepareInputForCurrentStep()
		return m, textinput.Blink
	default:
		if i, ok := parseOptionShortcut(msg.String(), len(dbOptions)); ok {
			m.dbIndex = i
			selected := dbOptions[i]
			m.config.DBDriver = selected.value
			m.config.DatabaseURL = defaultDatabaseURL(selected.value)
			m.step = stepDatabaseURL
			m.err = ""
			m.prepareInputForCurrentStep()
			return m, textinput.Blink
		}
		m.err = fmt.Sprintf("Pressed key: %q. Use number keys or Enter.", msg.String())
		return m, nil
	}
}

func (m Model) updateChannelStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left", "k", "shift+tab":
		m.channelIndex = (m.channelIndex - 1 + len(channelOptions)) % len(channelOptions)
		m.err = ""
		return m, nil
	case "down", "right", "j", "tab":
		m.channelIndex = (m.channelIndex + 1) % len(channelOptions)
		m.err = ""
		return m, nil
	case "enter":
		selected := channelOptions[m.channelIndex]
		m.config.ChatChannel = selected.value
		m.step = stepModel
		m.err = ""
		m.prepareInputForCurrentStep()
		return m, textinput.Blink
	default:
		if i, ok := parseOptionShortcut(msg.String(), len(channelOptions)); ok {
			m.channelIndex = i
			selected := channelOptions[i]
			m.config.ChatChannel = selected.value
			m.step = stepModel
			m.err = ""
			m.prepareInputForCurrentStep()
			return m, textinput.Blink
		}
		m.err = fmt.Sprintf("Pressed key: %q. Use number keys or Enter.", msg.String())
		return m, nil
	}
}

func (m Model) updateInputStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(m.input.Value())
		if value == "" {
			m.err = "This field is required."
			return m, nil
		}

		switch m.step {
		case stepDatabaseURL:
			m.config.DatabaseURL = value
			m.step = stepChannel
		case stepModel:
			m.config.ChatModel = value
			m.step = stepAPIKey
		case stepAPIKey:
			m.config.APIKey = value
			m.step = stepReview
		}

		m.err = ""
		m.prepareInputForCurrentStep()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateReviewStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path, err := saveConfig(m.config)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.savePath = path
		m.err = ""
		m.step = stepSaved
	case "b":
		m = m.goBack()
	}
	return m, nil
}

func (m *Model) prepareInputForCurrentStep() {
	m.input.SetValue("")
	m.input.EchoMode = textinput.EchoNormal
	m.input.Placeholder = ""
	m.input.Focus()

	switch m.step {
	case stepDatabaseURL:
		m.input.Placeholder = defaultDatabaseURL(m.config.DBDriver)
		m.input.SetValue(m.config.DatabaseURL)
	case stepModel:
		m.input.Placeholder = "gpt-4.1-mini"
		m.input.SetValue(m.config.ChatModel)
	case stepAPIKey:
		m.input.Placeholder = "sk-..."
		m.input.EchoMode = textinput.EchoPassword
		m.input.EchoCharacter = '•'
	}
}

func (m Model) goBack() Model {
	m.err = ""
	switch m.step {
	case stepDatabaseURL:
		m.step = stepDBDriver
	case stepChannel:
		m.step = stepDatabaseURL
	case stepModel:
		m.step = stepChannel
	case stepAPIKey:
		m.step = stepModel
	case stepReview:
		m.step = stepAPIKey
	case stepSaved:
		m.step = stepReview
	}
	m.prepareInputForCurrentStep()
	return m
}

func (m Model) renderOptionStep(title, description string, options []option, selected int) string {
	lines := []string{
		sectionTitleStyle.Render(title),
		mutedStyle.Render(description),
		"",
	}

	for i, item := range options {
		cursor := "  "
		label := fmt.Sprintf("%d. %s", i+1, item.label)
		hint := mutedStyle.Render(item.hint)
		if i == selected {
			cursor = highlightStyle.Render("› ")
			label = selectedStyle.Render(label)
		}
		lines = append(lines, cursor+label)
		lines = append(lines, "   "+hint)
		lines = append(lines, "")
	}

	if m.err != "" {
		lines = append(lines, errorStyle.Render(m.err))
	}

	return cardStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderInputStep(title, description string) string {
	lines := []string{
		sectionTitleStyle.Render(title),
		mutedStyle.Render(description),
		"",
		m.input.View(),
	}
	if m.err != "" {
		lines = append(lines, "", errorStyle.Render(m.err))
	}
	return cardStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderReview() string {
	rows := []string{
		sectionTitleStyle.Render("Review your onboarding setup"),
		mutedStyle.Render("Press Enter to save this sample config. Press Esc to edit any previous step."),
		"",
		fmt.Sprintf("%s %s", highlightStyle.Render("DB"), m.config.DBDriver),
		fmt.Sprintf("%s %s", highlightStyle.Render("DSN"), m.config.DatabaseURL),
		fmt.Sprintf("%s %s", highlightStyle.Render("Channel"), m.config.ChatChannel),
		fmt.Sprintf("%s %s", highlightStyle.Render("Model"), m.config.ChatModel),
		fmt.Sprintf("%s %s", highlightStyle.Render("Key"), maskSecret(m.config.APIKey)),
	}
	if m.err != "" {
		rows = append(rows, "", errorStyle.Render(m.err))
	}
	return cardStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderSaved() string {
	rows := []string{
		sectionTitleStyle.Render("Configuration saved"),
		mutedStyle.Render("The CLI and API can now load this config file directly. Env vars still override these values."),
		"",
		fmt.Sprintf("%s %s", highlightStyle.Render("Path"), m.savePath),
		fmt.Sprintf("%s %s", highlightStyle.Render("DB"), m.config.DBDriver),
		fmt.Sprintf("%s %s", highlightStyle.Render("Channel"), m.config.ChatChannel),
		fmt.Sprintf("%s %s", highlightStyle.Render("Model"), m.config.ChatModel),
		"",
		helpStyle.Render("Press Enter to exit."),
	}
	return cardStyle.Render(strings.Join(rows, "\n"))
}

func parseOptionShortcut(raw string, optionCount int) (int, bool) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	value--
	if value < 0 || value >= optionCount {
		return 0, false
	}
	return value, true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
