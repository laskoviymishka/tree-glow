package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)


var (
	treeWidthRatio = 0.25

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true)

	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// previewMsg is sent when async preview rendering completes.
type previewMsg struct {
	path     string
	lines    []string
	rawLines []string // original file content
	isMd     bool
}

type model struct {
	root          *node
	visible       []*node
	cursor        int
	width         int
	height        int
	treeScroll    int
	previewScroll int
	showHidden    bool

	cachedPath       string
	cachedLines      []string
	cachedRawLines   []string // original file content, no rendering
	cachedIsMarkdown bool
	loadingPreview   bool

	// selection state
	selecting       bool
	selectStartY    int
	selectEndY      int
	selectionActive bool

	// edit mode
	editMode     bool
	editPath     string
	editTextarea textarea.Model
	editDirty    bool // has unsaved changes
}

func newModel(rootPath string) model {
	root := newRootNode(rootPath, false)
	return model{root: root, visible: flatten(root)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) layoutWidths() (treeOuterW, previewOuterW int) {
	treeOuterW = int(float64(m.width) * treeWidthRatio)
	if treeOuterW < 20 {
		treeOuterW = 20
	}
	previewOuterW = m.width - treeOuterW - 1
	if previewOuterW < 10 {
		previewOuterW = 10
	}
	return
}

// requestPreview kicks off async preview if needed. Returns a tea.Cmd or nil.
func (m *model) requestPreview() tea.Cmd {
	if m.cursor >= len(m.visible) {
		m.cachedPath = ""
		m.cachedLines = nil
		m.loadingPreview = false
		return nil
	}
	path := m.visible[m.cursor].path
	if path == m.cachedPath && m.cachedLines != nil && !m.loadingPreview {
		return nil
	}
	// Show loading state immediately
	m.cachedPath = path
	m.cachedLines = []string{dimStyle.Render("  Loading...")}
	m.loadingPreview = true
	m.previewScroll = 0

	_, previewOuterW := m.layoutWidths()
	pw := previewOuterW - 6
	if pw < 20 {
		pw = 20
	}

	return func() tea.Msg {
		ext := strings.ToLower(filepath.Ext(path))
		isMd := ext == ".md" || ext == ".markdown"

		// Read raw file content for clipboard
		var rawLines []string
		if data, err := os.ReadFile(path); err == nil {
			rawLines = strings.Split(string(data), "\n")
		}

		// Render for display
		result := renderPreview(path, pw)
		rendered := strings.Split(result, "\n")
		if !isMd {
			for i, line := range rendered {
				rendered[i] = ansi.Truncate(line, pw, "")
			}
		}
		return previewMsg{path: path, lines: rendered, rawLines: rawLines, isMd: isMd}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Edit mode — route everything to textarea except save/quit
	if m.editMode {
		return m.updateEditMode(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.cachedPath = ""
		m.cachedLines = nil
		m.cachedRawLines = nil
		m.selectionActive = false
		m.selecting = false

	case previewMsg:
		if msg.path == m.cachedPath {
			m.cachedLines = msg.lines
			m.cachedRawLines = msg.rawLines
			m.cachedIsMarkdown = msg.isMd
			m.loadingPreview = false
			m.selectionActive = false
			m.selecting = false
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.previewScroll = 0
				m.selectionActive = false
			}
		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.previewScroll = 0
				m.selectionActive = false
			}
		case "enter", "right", "l":
			if m.cursor < len(m.visible) && m.visible[m.cursor].isDir {
				m.visible[m.cursor].toggle(m.showHidden)
				m.visible = flatten(m.root)
			}
		case "left", "h":
			if m.cursor < len(m.visible) {
				n := m.visible[m.cursor]
				if n.isDir && n.expanded {
					n.toggle(m.showHidden)
					m.visible = flatten(m.root)
				} else if n.parent != nil && n.parent != m.root {
					for i, v := range m.visible {
						if v == n.parent {
							m.cursor = i
							break
						}
					}
				}
			}
		case "pgdown", "ctrl+d":
			m.previewScroll += 20
		case "pgup", "ctrl+u":
			m.previewScroll -= 20
		case ".":
			m.showHidden = !m.showHidden
			m.reloadTree()
		case "G":
			m.cursor = len(m.visible) - 1
			m.previewScroll = 0
		case "g":
			m.cursor = 0
			m.previewScroll = 0
		case "e":
			if m.cursor < len(m.visible) && !m.visible[m.cursor].isDir {
				return m, m.enterEditMode()
			}
		case "esc":
			m.selectionActive = false
			m.selecting = false
		}

	case tea.MouseMsg:
		treeW, _ := m.layoutWidths()
		inPreview := msg.X >= treeW+2 // past tree border + gap
		previewContentY := msg.Y - 2 + m.previewScroll // map screen Y to content line

		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			if msg.X < treeW {
				if m.cursor > 0 {
					m.cursor--
					m.previewScroll = 0
					m.selectionActive = false
				}
			} else {
				m.previewScroll -= 3
			}
		case msg.Button == tea.MouseButtonWheelDown:
			if msg.X < treeW {
				if m.cursor < len(m.visible)-1 {
					m.cursor++
					m.previewScroll = 0
					m.selectionActive = false
				}
			} else {
				m.previewScroll += 3
			}
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			if inPreview && previewContentY >= 0 && previewContentY < len(m.cachedLines) {
				// Start selection in preview
				m.selecting = true
				m.selectionActive = true
				m.selectStartY = previewContentY
				m.selectEndY = previewContentY
			} else if msg.X < treeW && len(m.visible) > 0 {
				// Click in tree
				m.selectionActive = false
				m.selecting = false
				row := msg.Y - 1
				idx := m.treeScroll + row
				if idx >= 0 && idx < len(m.visible) {
					m.cursor = idx
					m.previewScroll = 0
					if m.visible[m.cursor].isDir {
						m.visible[m.cursor].toggle(m.showHidden)
						m.visible = flatten(m.root)
					}
				}
			}
		case msg.Action == tea.MouseActionMotion && m.selecting:
			// Drag — extend selection
			if previewContentY >= 0 && previewContentY < len(m.cachedLines) {
				m.selectEndY = previewContentY
			}
		case msg.Action == tea.MouseActionRelease && m.selecting:
			m.selecting = false
			if !inPreview {
				// Released outside preview — cancel
				m.selectionActive = false
			} else {
				if previewContentY >= 0 && previewContentY < len(m.cachedLines) {
					m.selectEndY = previewContentY
				}
				go m.copyToClipboard() // async to avoid blocking UI
			}
		}
	}

	// cursor bounds
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	cmd := m.requestPreview()

	// clamp preview scroll
	innerH := m.height - 4 // border top/bottom + title + status
	if innerH < 1 {
		innerH = 1
	}
	maxScroll := len(m.cachedLines) - innerH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.previewScroll > maxScroll {
		m.previewScroll = maxScroll
	}
	if m.previewScroll < 0 {
		m.previewScroll = 0
	}

	// tree scroll
	treeH := m.height - 4
	if treeH < 1 {
		treeH = 1
	}
	if m.cursor < m.treeScroll {
		m.treeScroll = m.cursor
	}
	if m.cursor >= m.treeScroll+treeH {
		m.treeScroll = m.cursor - treeH + 1
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.editMode {
		return m.viewEditMode()
	}

	innerH := m.height - 3
	if innerH < 1 {
		innerH = 1
	}

	treeOuterW, previewOuterW := m.layoutWidths()
	treeInnerW := treeOuterW - 2
	previewInnerW := previewOuterW - 2
	if previewInnerW < 4 {
		previewInnerW = 4
	}

	// --- Tree: exactly innerH lines, each truncated to treeInnerW ---
	treeContent := buildExactLines(m.buildTreeLines(treeInnerW, innerH), innerH, treeInnerW)
	treePanel := borderStyle.Width(treeInnerW).Height(innerH).MaxWidth(treeOuterW).MaxHeight(innerH+2).Render(treeContent)

	// --- Preview: title takes 1 row from the panel area ---
	// To keep both panels the same outer height:
	// preview outer = border(2) + previewInnerH = treeOuterH = innerH + 2
	// so previewInnerH = innerH
	// But we also need a title row, so we steal 1 row from inner content:
	previewContentH := innerH - 1
	if previewContentH < 1 {
		previewContentH = 1
	}

	// Title
	previewTitle := ""
	if m.cursor < len(m.visible) {
		previewTitle = m.visible[m.cursor].path
	}
	scrollInfo := ""
	if len(m.cachedLines) > previewContentH {
		maxScroll := len(m.cachedLines) - previewContentH
		pct := 0
		if maxScroll > 0 {
			pct = (m.previewScroll * 100) / maxScroll
		}
		if pct > 100 {
			pct = 100
		}
		scrollInfo = fmt.Sprintf(" %d%%", pct)
	}
	titleText := truncate(previewTitle, previewInnerW-len(scrollInfo)-2) + scrollInfo

	// Preview content: exactly previewContentH lines, with selection highlight
	previewSlice := m.sliceWithSelection(m.previewScroll, previewContentH)

	// Build preview inner: title + content, total = innerH lines
	previewInner := titleStyle.Width(previewInnerW).Render(titleText) + "\n" + previewSlice
	previewPanel := borderStyle.Width(previewInnerW).Height(innerH).MaxWidth(previewOuterW).MaxHeight(innerH+2).Render(previewInner)

	// Compose
	main := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, " ", previewPanel)
	var left string
	if m.selectionActive {
		selStart, selEnd := m.selectionRange()
		left = fmt.Sprintf(" Copied %d lines  |  esc: clear", selEnd-selStart+1)
	} else {
		left = " j/k:nav  enter:open  h:back  .:hidden  ctrl+d/u:scroll  q:quit"
	}
	right := ""
	if len(m.visible) > 0 {
		right = fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.visible))
	}
	gap := m.width - len(left) - len(right)
	if gap < 0 {
		// terminal too narrow — truncate hints
		left = ansi.Truncate(left, m.width-len(right)-1, "…")
		gap = 0
	}
	status := statusStyle.Render(left + strings.Repeat(" ", gap) + right)

	full := main + "\n" + status

	// SAFETY: hard clip to exactly m.height rows x m.width cols
	return hardClip(full, m.width, m.height)
}

// buildExactLines takes rendered lines, pads/clips to exactly `count` lines,
// each truncated to `width`.
func buildExactLines(lines []string, count, width int) string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		if i < len(lines) {
			out[i] = ansi.Truncate(lines[i], width, "")
		}
	}
	return strings.Join(out, "\n")
}

// sliceExact extracts `count` lines starting at `offset`, pads to exact count,
// truncates each to `width`.
var selectHighlight = lipgloss.NewStyle().
	Background(lipgloss.Color("24")).
	Foreground(lipgloss.Color("255"))

func (m model) sliceWithSelection(offset, count int) string {
	selStart, selEnd := m.selectionRange()
	out := make([]string, count)
	for i := 0; i < count; i++ {
		srcIdx := offset + i
		if srcIdx >= 0 && srcIdx < len(m.cachedLines) {
			line := m.cachedLines[srcIdx]
			if m.selectionActive && srcIdx >= selStart && srcIdx <= selEnd {
				// Strip existing ANSI and re-render with highlight
				line = selectHighlight.Render(ansi.Strip(line))
			}
			out[i] = line
		}
	}
	return strings.Join(out, "\n")
}

func sliceExact(lines []string, offset, count int) string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		srcIdx := offset + i
		if srcIdx >= 0 && srcIdx < len(lines) {
			out[i] = lines[srcIdx]
		}
	}
	return strings.Join(out, "\n")
}

func hardClip(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	out := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out[i] = ansi.Truncate(lines[i], width, "")
		}
	}
	return strings.Join(out, "\n")
}

func (m model) buildTreeLines(width, height int) []string {
	lines := make([]string, 0, height)
	end := m.treeScroll + height
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := m.treeScroll; i < end; i++ {
		n := m.visible[i]
		indent := strings.Repeat("  ", n.depth)
		icon := n.icon()
		label := n.name
		if n.isDir {
			label += "/"
		}
		line := indent + icon + label
		if i == m.cursor {
			vis := ansi.StringWidth(line)
			if vis < width {
				line += strings.Repeat(" ", width-vis)
			}
			line = selectedStyle.Render(ansi.Truncate(line, width, ""))
		} else if n.isDir {
			line = dirStyle.Render(ansi.Truncate(line, width, ""))
		} else {
			line = fileStyle.Render(ansi.Truncate(line, width, ""))
		}
		lines = append(lines, line)
	}
	return lines
}

// copyToClipboard copies selected text to system clipboard.
// For code files: uses raw source lines (no line numbers, no ANSI).
// For markdown: uses rendered text stripped of ANSI (since glamour changes line structure).
// Safe to call from a goroutine.
func (m *model) copyToClipboard() {
	startY, endY := m.selectStartY, m.selectEndY
	if startY > endY {
		startY, endY = endY, startY
	}
	if startY < 0 {
		startY = 0
	}

	var text string
	if m.cachedIsMarkdown || len(m.cachedRawLines) == 0 {
		// Markdown or no raw lines: use rendered text stripped of ANSI
		if endY >= len(m.cachedLines) {
			endY = len(m.cachedLines) - 1
		}
		if startY > endY {
			return
		}
		var lines []string
		for i := startY; i <= endY; i++ {
			lines = append(lines, ansi.Strip(m.cachedLines[i]))
		}
		text = strings.Join(lines, "\n")
	} else {
		// Code: use raw source lines
		if endY >= len(m.cachedRawLines) {
			endY = len(m.cachedRawLines) - 1
		}
		if startY > endY {
			return
		}
		text = strings.Join(m.cachedRawLines[startY:endY+1], "\n")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}

// selectionRange returns the normalized (min, max) of the selection.
func (m model) selectionRange() (int, int) {
	s, e := m.selectStartY, m.selectEndY
	if s > e {
		s, e = e, s
	}
	return s, e
}

// --- Edit mode ---

func (m *model) enterEditMode() tea.Cmd {
	path := m.visible[m.cursor].path
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	_, previewOuterW := m.layoutWidths()
	editW := previewOuterW - 4
	editH := m.height - 5
	if editW < 20 {
		editW = 20
	}
	if editH < 3 {
		editH = 3
	}

	ta := textarea.New()
	ta.SetValue(string(data))
	ta.SetWidth(editW)
	ta.SetHeight(editH)
	ta.ShowLineNumbers = true
	ta.Focus()
	ta.CharLimit = 0 // no limit

	m.editMode = true
	m.editPath = path
	m.editTextarea = ta
	m.editDirty = false

	return ta.Focus()
}

func (m model) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, previewOuterW := m.layoutWidths()
		m.editTextarea.SetWidth(previewOuterW - 4)
		m.editTextarea.SetHeight(m.height - 5)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			// Save file
			content := m.editTextarea.Value()
			if err := os.WriteFile(m.editPath, []byte(content), 0644); err == nil {
				m.editDirty = false
			}
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Exit edit mode
			m.editMode = false
			m.cachedPath = "" // force preview reload
			m.cachedLines = nil
			m.cachedRawLines = nil
			return m, nil
		}
	}

	// Forward to textarea
	oldVal := m.editTextarea.Value()
	var cmd tea.Cmd
	m.editTextarea, cmd = m.editTextarea.Update(msg)
	if m.editTextarea.Value() != oldVal {
		m.editDirty = true
	}
	return m, cmd
}

var (
	editTitleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("166")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	editTitleCleanStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("34")).
				Foreground(lipgloss.Color("230")).
				Bold(true).
				Padding(0, 1)
)

func (m model) viewEditMode() string {
	treeOuterW, previewOuterW := m.layoutWidths()
	treeInnerW := treeOuterW - 2
	innerH := m.height - 3
	if innerH < 1 {
		innerH = 1
	}

	// Tree (same as normal view)
	treeContent := buildExactLines(m.buildTreeLines(treeInnerW, innerH), innerH, treeInnerW)
	treePanel := borderStyle.Width(treeInnerW).Height(innerH).MaxWidth(treeOuterW).MaxHeight(innerH+2).Render(treeContent)

	// Edit panel
	previewInnerW := previewOuterW - 2

	// Title bar
	indicator := "EDIT"
	titleSt := editTitleCleanStyle
	if m.editDirty {
		indicator = "EDIT *"
		titleSt = editTitleStyle
	}
	titleText := indicator + "  " + truncate(m.editPath, previewInnerW-len(indicator)-4)
	header := titleSt.Width(previewOuterW).Render(titleText)

	// Textarea
	editContent := m.editTextarea.View()
	editPanel := borderStyle.Width(previewInnerW).Height(innerH-1).MaxWidth(previewOuterW).MaxHeight(innerH+1).Render(editContent)

	previewPanel := lipgloss.JoinVertical(lipgloss.Left, header, editPanel)
	main := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, " ", previewPanel)

	// Status bar
	left := " ctrl+s:save  esc:cancel  ctrl+c:quit"
	right := ""
	if m.editDirty {
		right = " modified "
	}
	gap := m.width - len(left) - len(right)
	if gap < 0 {
		left = ansi.Truncate(left, m.width-len(right)-1, "…")
		gap = 0
	}
	status := statusStyle.Render(left + strings.Repeat(" ", gap) + right)

	full := main + "\n" + status
	return hardClip(full, m.width, m.height)
}

func (m *model) reloadTree() {
	var currentPath string
	if m.cursor < len(m.visible) {
		currentPath = m.visible[m.cursor].path
	}
	m.root = newRootNode(m.root.path, m.showHidden)
	m.visible = flatten(m.root)
	m.cachedPath = ""
	for i, n := range m.visible {
		if n.path == currentPath {
			m.cursor = i
			return
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxLen {
		return s
	}
	return ansi.Truncate(s, maxLen-1, "…")
}

