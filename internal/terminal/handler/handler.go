package handler

import (
	"strings"

	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gu "github.com/termkit/gama/internal/github/usecase"
	hdlgithubrepo "github.com/termkit/gama/internal/terminal/handler/ghrepository"
	hdlWorkflow "github.com/termkit/gama/internal/terminal/handler/ghworkflow"
	hdlworkflowhistory "github.com/termkit/gama/internal/terminal/handler/ghworkflowhistory"
	hdlinfo "github.com/termkit/gama/internal/terminal/handler/information"
	ts "github.com/termkit/gama/internal/terminal/style"
)

type model struct {
	TabsWithColor []string

	TabContent  []string
	currentTab  int
	isTabActive bool

	viewport          viewport.Model
	terminalSizeReady bool

	timer timer.Model

	// models
	modelInfo       tea.Model
	actualModelInfo *hdlinfo.ModelInfo

	modelGithubRepository       tea.Model
	actualModelGithubRepository *hdlgithubrepo.ModelGithubRepository

	modelWorkflow       tea.Model
	directModelWorkflow *hdlWorkflow.ModelGithubWorkflow

	modelWorkflowHistory       tea.Model
	directModelWorkflowHistory *hdlworkflowhistory.ModelGithubWorkflowHistory
}

func SetupTerminal(githubUseCase gu.UseCase) tea.Model {
	tabsWithColor := []string{"Info", "Repository", "Workflow History", "Workflow", "Trigger"}

	tabContent := []string{
		"Information Page",
		"Repository Page",
		"Workflow History Page",
		"Workflow Page",
		"Trigger Page",
	}

	// setup models
	hdlModelInfo := hdlinfo.SetupModelInfo(githubUseCase)
	hdlModelGithubRepository := hdlgithubrepo.SetupModelGithubRepository(githubUseCase)
	hdlModelWorkflowHistory := hdlworkflowhistory.SetupModelGithubWorkflowHistory(githubUseCase)
	hdlModelWorkflow := hdlWorkflow.SetupModelGithubWorkflow(githubUseCase)

	m := model{TabsWithColor: tabsWithColor,
		TabContent: tabContent,
		timer:      timer.New(1<<63 - 1),
		modelInfo:  hdlModelInfo, actualModelInfo: hdlModelInfo,
		modelGithubRepository: hdlModelGithubRepository, actualModelGithubRepository: hdlModelGithubRepository,
		modelWorkflowHistory: hdlModelWorkflowHistory, directModelWorkflowHistory: hdlModelWorkflowHistory,
		modelWorkflow: hdlModelWorkflow, directModelWorkflow: hdlModelWorkflow,
	}

	hdlModelInfo.Viewport = &m.viewport
	hdlModelGithubRepository.Viewport = &m.viewport
	hdlModelWorkflowHistory.Viewport = &m.viewport
	hdlModelWorkflow.Viewport = &m.viewport

	return &m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.timer.Init(), m.modelInfo.Init(), m.modelGithubRepository.Init())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Sync terminal size
	m.syncTerminal(msg)

	var cmds []tea.Cmd

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "right":
			m.currentTab = min(m.currentTab+1, len(m.TabsWithColor)-1)
			cmds = append(cmds, m.handleTabContent(cmd, msg))
		case "left":
			m.currentTab = max(m.currentTab-1, 0)
			cmds = append(cmds, m.handleTabContent(cmd, msg))
		case "enter":
			cmds = append(cmds, m.handleTabContent(cmd, msg))
		case "esc", "z":
			cmds = append(cmds, m.handleTabContent(cmd, msg))
		case "ctrl+c", "q":
			return m, tea.Quit
		default:
			cmds = append(cmds, m.handleTabContent(cmd, msg))
		}
	case timer.TickMsg:
		m.timer, cmd = m.timer.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if !m.terminalSizeReady {
		return "Setting up..."
	}
	if m.viewport.Width < 80 || m.viewport.Height < 24 {
		return "Terminal window is too small. Please resize to at least 80x24."
	}

	var mainDoc strings.Builder
	var helpDoc string
	var operationDoc string
	var helpDocHeight int

	var renderedTabs []string
	for i, t := range m.TabsWithColor {
		var style lipgloss.Style
		isActive := i == m.currentTab
		if isActive {
			style = ts.TitleStyleActive.Copy()
		} else {
			style = ts.TitleStyleDisable.Copy()
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	mainDoc.WriteString(m.headerView(renderedTabs...) + "\n")

	var width = lipgloss.Width(strings.Repeat("-", m.viewport.Width)) - len(renderedTabs)
	dynamicWindowStyle := ts.WindowStyleCyan.Width(width).Height(m.viewport.Height - 20)

	helpWindowStyle := ts.WindowStyleHelp.Width(width)
	operationWindowStyle := lipgloss.NewStyle()

	switch m.currentTab {
	case 0:
		mainDoc.WriteString(dynamicWindowStyle.Render(m.modelInfo.View()))

		if m.actualModelInfo.IsError() {
			operationWindowStyle = ts.WindowStyleError.Width(width)
		} else {
			operationWindowStyle = ts.WindowStyleOperation.Width(width)
		}
		operationDoc = operationWindowStyle.Render(m.actualModelInfo.ViewErrorOrOperation())

		helpDoc = helpWindowStyle.Render(m.actualModelInfo.ViewHelp())
	case 1:
		mainDoc.WriteString(dynamicWindowStyle.Render(m.modelGithubRepository.View()))

		if m.actualModelGithubRepository.IsError() {
			operationWindowStyle = ts.WindowStyleError.Width(width)
		} else {
			operationWindowStyle = ts.WindowStyleOperation.Width(width)
		}
		operationDoc = operationWindowStyle.Render(m.actualModelGithubRepository.ViewErrorOrOperation())

		helpDoc = helpWindowStyle.Render(m.actualModelGithubRepository.ViewHelp())
	case 2:
		mainDoc.WriteString(dynamicWindowStyle.Render(m.modelWorkflowHistory.View()))

		if m.directModelWorkflowHistory.IsError() {
			operationWindowStyle = ts.WindowStyleError.Width(width)
		} else {
			operationWindowStyle = ts.WindowStyleOperation.Width(width)
		}
		operationDoc = operationWindowStyle.Render(m.directModelWorkflowHistory.ViewErrorOrOperation())

		helpDoc = helpWindowStyle.Render(m.directModelWorkflowHistory.ViewHelp())
	case 3:
		mainDoc.WriteString(dynamicWindowStyle.Render(m.modelWorkflow.View()))

		if m.directModelWorkflow.IsError() {
			operationWindowStyle = ts.WindowStyleError.Width(width)
		} else {
			operationWindowStyle = ts.WindowStyleOperation.Width(width)
		}
		operationDoc = operationWindowStyle.Render(m.directModelWorkflow.ViewErrorOrOperation())

		helpDoc = helpWindowStyle.Render(m.directModelWorkflow.ViewHelp())
	case 4:
		mainDoc.WriteString(dynamicWindowStyle.Render("Trigger Page\n"))
	}

	mainDocContent := ts.DocStyle.Render(mainDoc.String())

	mainDocHeight := strings.Count(mainDocContent, "\n")
	helpDocHeight = strings.Count(helpDoc, "\n")
	errorDocHeight := strings.Count(operationDoc, "\n")
	requiredNewlinesForPadding := m.viewport.Height - mainDocHeight - helpDocHeight - errorDocHeight - 1
	padding := strings.Repeat("\n", max(0, requiredNewlinesForPadding))

	pageInformation := lipgloss.JoinVertical(lipgloss.Top, operationDoc, helpDoc)

	return mainDocContent + padding + pageInformation
}

func (m *model) syncTerminal(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())

		if !m.terminalSizeReady {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.viewport.YPosition = headerHeight
			m.terminalSizeReady = true
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
	}
}

func (m *model) handleTabContent(cmd tea.Cmd, msg tea.Msg) tea.Cmd {
	switch m.currentTab {
	case 0:
		m.modelInfo, cmd = m.modelInfo.Update(msg)
	case 1:
		m.modelGithubRepository, cmd = m.modelGithubRepository.Update(msg)
	case 2:
		m.modelWorkflowHistory, cmd = m.modelWorkflowHistory.Update(msg)
	case 3:
		m.modelWorkflow, cmd = m.modelWorkflow.Update(msg)
	}
	return cmd
}

func (m *model) headerView(titles ...string) string {
	var renderedTitles string
	for _, t := range titles {
		renderedTitles += t
	}
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(renderedTitles)))
	titles = append(titles, line)
	return lipgloss.JoinHorizontal(lipgloss.Center, titles...)
}