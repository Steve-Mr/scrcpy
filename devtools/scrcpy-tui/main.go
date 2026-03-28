package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Genymobile/scrcpy/devtools/scrcpy-tui/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type state int

const (
	stateDeviceSelection state = iota
	stateDashboard
	stateAppSelection
	statePresetSelection
	stateInput
)

type dashboardOption int

const (
	optResolution dashboardOption = iota
	optNewDisplay
	optFPS
	optBitrate
	optCodec
	optApp
	optPreset
	optRun
	optAddToPath
)

type model struct {
	state          state
	devices        []string
	selectedDevice string
	apps           []string
	selectedApp    string

	presets        []config.Preset
	selectedPreset *config.Preset

	// Dashboard options
	maxResolution string
	newDisplay    string
	maxFPS        string
	bitRate       string
	codec         string

	// List bubble
	list list.Model

	// Input bubble
	input textinput.Model

	// Dashboard state
	cursor dashboardOption

	width, height int
	err           error
	scrcpyPath    string
	adbPath       string
}

func findBinaries() (string, string) {
	cwd, _ := os.Getwd()
	scrcpy := "scrcpy"
	adb := "adb"

	// Add .exe on Windows
	scrcpyExe := "scrcpy"
	adbExe := "adb"
	if os.PathSeparator == '\\' {
		scrcpyExe += ".exe"
		adbExe += ".exe"
	}

	if _, err := os.Stat(filepath.Join(cwd, scrcpyExe)); err == nil {
		scrcpy = filepath.Join(cwd, scrcpyExe)
	}
	if _, err := os.Stat(filepath.Join(cwd, adbExe)); err == nil {
		adb = filepath.Join(cwd, adbExe)
	}
	return scrcpy, adb
}

func getDevices(adbPath string) ([]string, error) {
	cmd := exec.Command(adbPath, "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "device") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}
	return devices, nil
}

func getApps(scrcpyPath, adbPath, device string) ([]string, error) {
	cmd := exec.Command(scrcpyPath, "-s", device, "--list-apps")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to adb if scrcpy fails to list apps
		cmd = exec.Command(adbPath, "-s", device, "shell", "pm", "list", "packages")
		output, _ = cmd.CombinedOutput()
	}

	var apps []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.Contains(line, "package:") {
			apps = append(apps, strings.TrimPrefix(line, "package:"))
		} else if (strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")) && len(line) > 2 {
			// scrcpy --list-apps output format:
			//  - App Name                      com.example.app
			//  * System App Name               com.android.system
			content := line[2:]
			// The package name is at the end of the line
			parts := strings.Fields(content)
			if len(parts) >= 1 {
				apps = append(apps, parts[len(parts)-1])
			}
		}
	}
	return apps, nil
}

func initialModel() model {
	scrcpy, adb := findBinaries()
	cfg, _ := config.LoadConfig("config.yaml")

	ti := textinput.New()
	ti.Focus()

	return model{
		state:      stateDeviceSelection,
		scrcpyPath: scrcpy,
		adbPath:    adb,
		presets:    cfg.Presets,
		codec:      "h264",
		maxFPS:     "60",
		bitRate:    "8M",
		input:      ti,
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		devices, err := getDevices(m.adbPath)
		if err != nil {
			return err
		}
		return devicesMsg(devices)
	}
}

type devicesMsg []string
type appsMsg []string

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state != stateInput {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-4)

	case devicesMsg:
		m.devices = msg
		items := make([]list.Item, len(m.devices))
		for i, d := range m.devices {
			items[i] = item{title: d, desc: "Android Device"}
		}
		m.list = list.New(items, list.NewDefaultDelegate(), m.width-4, m.height-4)
		m.list.Title = "Select Device"
		return m, nil

	case appsMsg:
		m.apps = msg
		items := make([]list.Item, len(m.apps))
		for i, a := range m.apps {
			items[i] = item{title: a, desc: "Package Name"}
		}
		m.list = list.New(items, list.NewDefaultDelegate(), m.width-4, m.height-4)
		m.list.Title = "Select App to Start"
		m.state = stateAppSelection
		return m, nil

	case error:
		m.err = msg
		return m, nil
	}

	switch m.state {
	case stateDeviceSelection:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selectedDevice = i.title
				m.state = stateDashboard
			}
		}
		return m, cmd

	case stateDashboard:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < optAddToPath {
					m.cursor++
				}
			case "enter":
				switch m.cursor {
				case optResolution:
					m.state = stateInput
					m.input.Placeholder = "e.g. 1024"
					m.input.SetValue(m.maxResolution)
				case optNewDisplay:
					m.state = stateInput
					m.input.Placeholder = "e.g. 1920x1080"
					m.input.SetValue(m.newDisplay)
				case optFPS:
					m.state = stateInput
					m.input.Placeholder = "e.g. 60"
					m.input.SetValue(m.maxFPS)
				case optBitrate:
					m.state = stateInput
					m.input.Placeholder = "e.g. 8M"
					m.input.SetValue(m.bitRate)
				case optCodec:
					m.state = stateInput
					m.input.Placeholder = "h264, h265, or av1"
					m.input.SetValue(m.codec)
				case optApp:
					return m, func() tea.Msg {
						apps, err := getApps(m.scrcpyPath, m.adbPath, m.selectedDevice)
						if err != nil {
							return err
						}
						return appsMsg(apps)
					}
				case optPreset:
					items := make([]list.Item, len(m.presets))
					for i, p := range m.presets {
						items[i] = item{title: p.Name, desc: fmt.Sprintf("Res: %s, FPS: %d", p.MaxResolution, p.MaxFPS)}
					}
					m.list = list.New(items, list.NewDefaultDelegate(), m.width-4, m.height-4)
					m.list.Title = "Select Preset"
					m.state = statePresetSelection
				case optRun:
					return m, m.runScrcpy()
				case optAddToPath:
					return m, m.addToPath()
				}
			}
		}

	case stateAppSelection:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selectedApp = i.title
				m.state = stateDashboard
			}
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = stateDashboard
		}
		return m, cmd

	case statePresetSelection:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				for _, p := range m.presets {
					if p.Name == i.title {
						m.selectedPreset = &p
						m.maxResolution = p.MaxResolution
						m.maxFPS = fmt.Sprintf("%d", p.MaxFPS)
						m.codec = p.VideoCodec
						m.bitRate = p.VideoBitRate
						if p.AppBindings != nil {
							// If there's an app binding, select the first one for now
							// or we could show another list if there are multiple.
							for _, pkg := range p.AppBindings {
								m.selectedApp = pkg
								break
							}
						}
						break
					}
				}
				m.state = stateDashboard
			}
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = stateDashboard
		}
		return m, cmd

	case stateInput:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			val := m.input.Value()
			switch m.cursor {
			case optResolution:
				m.maxResolution = val
			case optNewDisplay:
				m.newDisplay = val
			case optFPS:
				m.maxFPS = val
			case optBitrate:
				m.bitRate = val
			case optCodec:
				m.codec = val
			}
			m.state = stateDashboard
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = stateDashboard
		}
		return m, cmd
	}

	return m, nil
}

func (m model) addToPath() tea.Cmd {
	return func() tea.Msg {
		cwd, _ := os.Getwd()
		home, _ := os.UserHomeDir()
		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = filepath.Join(home, ".zshrc")
		} else {
			rcFile = filepath.Join(home, ".bashrc")
		}

		data, err := os.ReadFile(rcFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		exportLine := fmt.Sprintf("export PATH=\"$PATH:%s\"", cwd)
		if strings.Contains(string(data), exportLine) {
			return nil // Already in PATH
		}

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + exportLine + "\n"); err != nil {
			return err
		}
		return nil
	}
}

func (m model) runScrcpy() tea.Cmd {
	args := m.generateArgs()
	return tea.ExecProcess(exec.Command(m.scrcpyPath, args...), func(err error) tea.Msg {
		return err
	})
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress q to quit.", m.err)
	}

	switch m.state {
	case stateDeviceSelection, stateAppSelection, statePresetSelection:
		return docStyle.Render(m.list.View())
	case stateDashboard:
		s := titleStyle.Render("scrcpy TUI Dashboard") + "\n\n"
		s += fmt.Sprintf("Device: %s\n\n", m.selectedDevice)

		options := []string{
			fmt.Sprintf("Max Resolution: %s", m.maxResolution),
			fmt.Sprintf("New Display: %s", m.newDisplay),
			fmt.Sprintf("Max FPS: %s", m.maxFPS),
			fmt.Sprintf("Bitrate: %s", m.bitRate),
			fmt.Sprintf("Codec: %s", m.codec),
			fmt.Sprintf("App: %s", m.selectedApp),
			fmt.Sprintf("Preset: %s", func() string {
				if m.selectedPreset != nil {
					return m.selectedPreset.Name
				}
				return "None"
			}()),
			"RUN",
			"Add current dir to PATH",
		}

		for i, opt := range options {
			cursor := " "
			if m.cursor == dashboardOption(i) {
				cursor = ">"
				s += cursorStyle.Render(fmt.Sprintf("%s %s\n", cursor, opt))
			} else {
				s += fmt.Sprintf("%s %s\n", cursor, opt)
			}
		}

		s += "\n(Arrows to move, Enter to select/edit, q to quit)\n"

		// Command preview
		s += "\nCommand Preview:\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.generateCommandString())

		return docStyle.Render(s)
	case stateInput:
		return docStyle.Render(fmt.Sprintf(
			"Enter value for %s:\n\n%s\n\n(Enter to confirm, Esc to cancel)",
			[]string{"Max Resolution", "New Display", "Max FPS", "Bitrate", "Codec"}[m.cursor],
			m.input.View(),
		))
	default:
		return "Unknown state"
	}
}

func (m model) generateArgs() []string {
	args := []string{"-s", m.selectedDevice}
	if m.maxResolution != "" {
		args = append(args, "-m", m.maxResolution)
	}
	if m.newDisplay != "" {
		args = append(args, "--new-display", m.newDisplay)
	}
	if m.maxFPS != "" {
		args = append(args, "--max-fps", m.maxFPS)
	}
	if m.bitRate != "" {
		args = append(args, "-b", m.bitRate)
	}
	if m.codec != "" {
		args = append(args, "--video-codec", m.codec)
	}
	if m.selectedApp != "" {
		args = append(args, "--start-app", m.selectedApp)
	}

	if m.selectedPreset != nil {
		args = append(args, m.selectedPreset.Args...)
	}
	return args
}

func (m model) generateCommandString() string {
	args := m.generateArgs()
	return fmt.Sprintf("%s %s", m.scrcpyPath, strings.Join(args, " "))
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
