package cmd

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	green = lipgloss.Color("#00f63e")
	red   = lipgloss.Color("#ff4d4d")
	white = lipgloss.Color("#e2e2e2")
	gray  = lipgloss.Color("#474747")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(gray).
			MarginBottom(1).
			PaddingBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	fileStyle = lipgloss.NewStyle().
			Foreground(white)

	dimStyle = lipgloss.NewStyle().
			Foreground(gray)

	addedStyle = lipgloss.NewStyle().
			Foreground(green)

	removedStyle = lipgloss.NewStyle().
			Foreground(red)

	headerStyle = lipgloss.NewStyle().
			Foreground(gray).
			Italic(true)

	selectedFileStyle = lipgloss.NewStyle().
				Foreground(green).
				Bold(true)
)

var commitTypes = []struct {
	label       string
	description string
}{
	{"feat", "a new feature"},
	{"fix", "a bug fix"},
	{"chore", "maintenance, tooling, config"},
	{"docs", "documentation changes"},
	{"refactor", "code restructure, no behavior change"},
	{"style", "formatting, missing semicolons, etc"},
	{"test", "adding or updating tests"},
	{"perf", "performance improvements"},
}

type model struct {
	files         []string        // all files (staged + unstaged) for display
	fileMap       map[string]bool // true = already git-staged; never mutated except after ctrl+s succeeds
	cursor        int             // current cursor position in files list
	diff          string          // diff for files[cursor]
	screen        string          // current screen: files, diff, type, message, confirm, done
	commitType    string          // selected commit type
	typeCursor    int             // cursor for commit type selection
	commitMessage string          // typed commit message
	commitError   string          // error message on commit failure
	diffScroll    int             // vertical scroll in diff screen
	height        int             // terminal height
	tempStaging   []string        // unstaged files queued to be staged on ctrl+s
}

func (m model) isEffectivelyStaged(file string) bool {
	return m.fileMap[file] || slices.Contains(m.tempStaging, file)
}

func setInitialModel(files []string, fileMap map[string]bool) model {
	return model{
		files:       files,
		fileMap:     fileMap,
		cursor:      0,
		screen:      "files",
		tempStaging: []string{},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.screen != "message" {
				switch m.screen {
				case "files":
					if m.cursor > 0 {
						m.cursor--
					}
				case "type":
					if m.typeCursor > 0 {
						m.typeCursor--
					}
				case "diff":
					if m.diffScroll > 0 {
						m.diffScroll--
					}
				}
			}
		case "down":
			if m.screen != "message" {
				switch m.screen {
				case "files":
					if m.cursor < len(m.files)-1 {
						m.cursor++
					}
				case "type":
					if m.typeCursor < len(commitTypes)-1 {
						m.typeCursor++
					}
				case "diff":
					lines := strings.Split(strings.TrimSpace(m.diff), "\n")
					visibleLines := m.height - 8
					if m.diffScroll < len(lines)-visibleLines {
						m.diffScroll++
					}
				}
			}
		case "enter":
			switch m.screen {
			case "files":
				file := m.files[m.cursor]
				diff, err := getFileDiff(file, m.fileMap[file])
				if err != nil {
					diff = "Could not read diff for " + file
				}
				m.diff = diff
				m.diffScroll = 0
				m.screen = "diff"
			case "diff":
				file := m.files[m.cursor]
				if m.fileMap[file] {
				} else if slices.Contains(m.tempStaging, file) {
					m.tempStaging = slices.DeleteFunc(m.tempStaging, func(f string) bool {
						return f == file
					})
				} else {
					m.tempStaging = append(m.tempStaging, file)
				}
				m.screen = "files"
			case "type":
				m.commitType = commitTypes[m.typeCursor].label
				m.screen = "message"
			case "message":
				if m.commitMessage != "" {
					m.screen = "confirm"
				}
			case "confirm":
				finalCommit := fmt.Sprintf("%s: %s", m.commitType, m.commitMessage)
				if err := runCommit(finalCommit); err != nil {
					m.commitError = err.Error()
				}
				m.screen = "done"
			}
		case "esc":
			switch m.screen {
			case "diff":
				m.screen = "files"
			case "type":
				m.screen = "diff"
			case "message":
				m.screen = "type"
			case "confirm":
				m.screen = "message"
			}
		case "backspace":
			if m.screen == "message" && len(m.commitMessage) > 0 {
				m.commitMessage = m.commitMessage[:len(m.commitMessage)-1]
			}
		case "ctrl+a":
			if m.screen == "files" {
				for _, f := range m.files {
					if !m.fileMap[f] && !slices.Contains(m.tempStaging, f) {
						m.tempStaging = append(m.tempStaging, f)
					}
				}
			}
		case "ctrl+x":
			if m.screen == "files" {
				m.tempStaging = []string{}
			}
		case "ctrl+s":
			if m.screen == "files" && len(m.tempStaging) > 0 {
				if err := addFilesToStaging(m.tempStaging); err != nil {
					fmt.Printf("Error staging files: %s\n", err)
				} else {
					for _, f := range m.tempStaging {
						m.fileMap[f] = true
					}
					m.tempStaging = []string{}
				}
			}
		default:
			if m.screen == "message" {
				if len(msg.String()) == 1 {
					m.commitMessage += msg.String()
				}
			} else if msg.String() == "q" {
				return m, tea.Quit
			} else if msg.String() == "c" && m.screen == "files" {
				hasStaged, err := checkForStagedFiles()
				if err != nil {
					fmt.Printf("Could not start commit: %s", err)
				}
				if hasStaged {
					m.screen = "type"
				}
			} else if msg.String() == "b" {
				switch m.screen {
				case "diff":
					m.screen = "files"
				case "type":
					m.screen = "diff"
				case "confirm":
					m.screen = "message"
				}
			} else if msg.String() == "j" {
				switch m.screen {
				case "files":
					if m.cursor < len(m.files)-1 {
						m.cursor++
					}
				case "type":
					if m.typeCursor < len(commitTypes)-1 {
						m.typeCursor++
					}
				case "diff":
					lines := strings.Split(strings.TrimSpace(m.diff), "\n")
					visibleLines := m.height - 8
					if m.diffScroll < len(lines)-visibleLines {
						m.diffScroll++
					}
				}
			} else if msg.String() == "k" {
				switch m.screen {
				case "files":
					if m.cursor > 0 {
						m.cursor--
					}
				case "type":
					if m.typeCursor > 0 {
						m.typeCursor--
					}
				case "diff":
					if m.diffScroll > 0 {
						m.diffScroll--
					}
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case "diff":
		return viewDiff(m)
	case "type":
		return viewCommitType(m)
	case "message":
		return viewMessage(m)
	case "confirm":
		return viewConfirm(m)
	case "done":
		return viewDone(m)
	default:
		return viewFiles(m)
	}
}

func viewFiles(m model) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("AUTOCOMMIT\nv1.0.0\n@migwell-dev"))
	s.WriteString("\n")

	allQueued := len(m.files) > 0
	for _, f := range m.files {
		if !m.fileMap[f] && !slices.Contains(m.tempStaging, f) {
			allQueued = false
			break
		}
	}

	if len(m.tempStaging) > 0 {
		if allQueued {
			s.WriteString(dimStyle.Render("↑↓/j,k navigate · enter view diff · ctrl+x clear staging queue · ctrl+s stage files · q quit"))
		} else {
			s.WriteString(dimStyle.Render("↑↓/j,k navigate · enter view diff · ctrl+a add all · ctrl+x clear staging queue · ctrl+s stage files · q quit"))
		}
	} else {
		s.WriteString(dimStyle.Render("↑↓/j,k navigate · enter view diff · ctrl+a add all files to staging queue · c start commit · q quit"))
	}
	s.WriteString("\n")
	for i, file := range m.files {
		var display string
		switch {
		case m.fileMap[file]:
			display = file + " (staged)"
		case slices.Contains(m.tempStaging, file):
			display = file + " (will be staged)"
		default:
			display = file
		}

		if m.cursor == i {
			row := lipgloss.NewStyle().
				Background(lipgloss.Color("#1f1f1f")).
				Foreground(green).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(4).
				Render("▌ " + display)
			s.WriteString(row)
		} else {
			row := lipgloss.NewStyle().
				Foreground(white).
				PaddingLeft(1).
				Render("  " + display)
			s.WriteString(row)
		}
		s.WriteString("\n")
	}

	return s.String()
}

func viewDiff(m model) string {
	var s strings.Builder

	file := m.files[m.cursor]
	isGitStaged := m.fileMap[file]
	isTempStaged := slices.Contains(m.tempStaging, file)

	var hint string
	switch {
	case isGitStaged:
		hint = "j/k scroll · already staged · esc/b go back"
	case isTempStaged:
		hint = "j/k scroll · enter to un-queue · esc/b go back"
	default:
		hint = "j/k scroll · enter to queue for staging · esc/b go back"
	}

	s.WriteString(titleStyle.Render(file) + "\n")
	s.WriteString(dimStyle.Render(hint) + "\n\n")

	lines := strings.Split(strings.TrimSpace(m.diff), "\n")
	visibleLines := m.height - 8
	if visibleLines < 1 {
		visibleLines = 10
	}
	end := min(m.diffScroll+visibleLines, len(lines))

	for i, line := range lines[m.diffScroll:end] {
		var rendered string
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			rendered = headerStyle.Render(line)
		case strings.HasPrefix(line, "+"):
			rendered = addedStyle.Render(line)
		case strings.HasPrefix(line, "-"):
			rendered = removedStyle.Render(line)
		case strings.HasPrefix(line, "@@"):
			rendered = dimStyle.Render(line)
		default:
			rendered = fileStyle.Render(line)
		}

		if i == 0 {
			row := lipgloss.NewStyle().
				Background(lipgloss.Color("#1f1f1f")).
				PaddingRight(4).
				Render("▌ " + rendered)
			s.WriteString(row)
		} else {
			s.WriteString("  " + rendered)
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	var status string
	switch {
	case isGitStaged:
		status = "git-staged"
	case isTempStaged:
		status = "queued"
	default:
		status = "unstaged"
	}
	s.WriteString(dimStyle.Render(fmt.Sprintf("lines %d–%d of %d · %s", m.diffScroll+1, end, len(lines), status)))

	return s.String()
}

func viewCommitType(m model) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("COMMIT TYPE"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("↑↓/j,k navigate · enter select · esc/b go back"))
	s.WriteString("\n\n")

	for i, ct := range commitTypes {
		if m.typeCursor == i {
			row := lipgloss.NewStyle().
				Background(lipgloss.Color("#1f1f1f")).
				Foreground(green).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(4).
				Render("▌ " + ct.label)
			s.WriteString(row)
			s.WriteString("  ")
			s.WriteString(dimStyle.Render(ct.description))
		} else {
			row := lipgloss.NewStyle().
				Foreground(white).
				PaddingLeft(1).
				Render("  " + ct.label)
			s.WriteString(row)
			s.WriteString("  ")
			s.WriteString(dimStyle.Render(ct.description))
		}
		s.WriteString("\n")
	}

	return s.String()
}

func viewMessage(m model) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("COMMIT MESSAGE"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("type your message · enter continue · esc/b go back"))
	s.WriteString("\n\n")

	s.WriteString(dimStyle.Render("type: "))
	s.WriteString(selectedFileStyle.Render(m.commitType))
	s.WriteString("\n\n")

	s.WriteString(fileStyle.Render("Description: "))
	s.WriteString(selectedFileStyle.Render(m.commitMessage))
	s.WriteString(cursorStyle.Render("▌"))
	s.WriteString("\n\n")

	if m.commitMessage == "" {
		s.WriteString(dimStyle.Render("(cannot be empty)"))
	}

	return s.String()
}

func viewConfirm(m model) string {
	var s strings.Builder

	finalCommit := fmt.Sprintf("%s: %s", m.commitType, m.commitMessage)

	s.WriteString(titleStyle.Render("CONFIRM COMMIT"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("enter to commit · esc/b go back · ctrl+c quit"))
	s.WriteString("\n\n")

	s.WriteString(dimStyle.Render("Your commit message will be:"))
	s.WriteString("\n\n")

	s.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 2).
		Render(selectedFileStyle.Render(finalCommit)))

	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("Staged files included:"))
	s.WriteString("\n")
	for _, f := range m.files {
		if m.isEffectivelyStaged(f) {
			s.WriteString(dimStyle.Render("  - " + f))
			s.WriteString("\n")
		}
	}

	return s.String()
}

func viewDone(m model) string {
	var s strings.Builder

	if m.commitError != "" {
		s.WriteString(titleStyle.Render("COMMIT FAILED"))
		s.WriteString("\n\n")
		s.WriteString(removedStyle.Render("Error: " + m.commitError))
		s.WriteString("\n\n")
		s.WriteString(dimStyle.Render("press ctrl+c to exit"))
		return s.String()
	}

	finalCommit := fmt.Sprintf("%s: %s", m.commitType, m.commitMessage)

	s.WriteString(titleStyle.Render("COMMITTED"))
	s.WriteString("\n\n")
	s.WriteString(addedStyle.Render("✓ success"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 2).
		Render(selectedFileStyle.Render(finalCommit)))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("press ctrl+c to exit"))

	return s.String()
}
