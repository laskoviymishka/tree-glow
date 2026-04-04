package main

import (
	"fmt"
	"path/filepath"
	"strings"

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

type model struct {
	root          *node
	visible       []*node
	cursor        int
	width         int
	height        int
	treeScroll    int
	previewScroll int
	showHidden    bool

	cachedPath    string
	cachedLines   []string
	cachedIsMarkdown bool
}

func newModel(rootPath string) model {
	root := newRootNode(rootPath, false)
	return model{root: root, visible: flatten(root)}
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) refreshPreviewCache() {
	if m.cursor >= len(m.visible) {
		m.cachedPath = ""
		m.cachedLines = nil
		return
	}
	path := m.visible[m.cursor].path
	if path == m.cachedPath && m.cachedLines != nil {
		return
	}
	treeW := int(float64(m.width) * treeWidthRatio)
	if treeW < 20 {
		treeW = 20
	}
	pw := m.width - treeW - 8
	if pw < 20 {
		pw = 20
	}
	ext := strings.ToLower(filepath.Ext(path))
	m.cachedIsMarkdown = ext == ".md" || ext == ".markdown"
	result := renderPreview(path, pw)
	m.cachedPath = path
	raw := strings.Split(result, "\n")
	if m.cachedIsMarkdown {
		m.cachedLines = raw
	} else {
		// Pre-truncate code lines to prevent lipgloss word-wrap
		m.cachedLines = make([]string, len(raw))
		for i, line := range raw {
			m.cachedLines[i] = ansi.Truncate(line, pw, "")
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.cachedPath = ""

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.previewScroll = 0
			}
		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.previewScroll = 0
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
		}

	case tea.MouseMsg:
		treeW := int(float64(m.width) * treeWidthRatio)
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if msg.X < treeW {
				if m.cursor > 0 {
					m.cursor--
					m.previewScroll = 0
				}
			} else {
				m.previewScroll -= 3
			}
		case tea.MouseButtonWheelDown:
			if msg.X < treeW {
				if m.cursor < len(m.visible)-1 {
					m.cursor++
					m.previewScroll = 0
				}
			} else {
				m.previewScroll += 3
			}
		case tea.MouseButtonLeft:
			if msg.X < treeW {
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
		}
	}

	// cursor bounds
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	m.refreshPreviewCache()

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

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Layout budget:
	//   border top + border bottom = 2 rows
	//   status bar = 1 row
	//   inner content = m.height - 3 rows
	// Both panels share the same outer height = m.height - 1 (minus status)
	// Inner height for both = m.height - 3

	innerH := m.height - 3
	if innerH < 1 {
		innerH = 1
	}

	treeOuterW := int(float64(m.width) * treeWidthRatio)
	if treeOuterW < 20 {
		treeOuterW = 20
	}
	treeInnerW := treeOuterW - 2 // border left + right

	previewOuterW := m.width - treeOuterW - 1 // 1 gap
	previewInnerW := previewOuterW - 2

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
		bottomLine := m.previewScroll + previewContentH
		pct := (bottomLine * 100) / len(m.cachedLines)
		if pct > 100 {
			pct = 100
		}
		scrollInfo = fmt.Sprintf(" %d%%", pct)
	}
	titleText := truncate(previewTitle, previewInnerW-len(scrollInfo)-2) + scrollInfo

	// Preview content: exactly previewContentH lines
	previewSlice := sliceExact(m.cachedLines, m.previewScroll, previewContentH)

	// Build preview inner: title + content, total = innerH lines
	previewInner := titleStyle.Width(previewInnerW).Render(titleText) + "\n" + previewSlice
	previewPanel := borderStyle.Width(previewInnerW).Height(innerH).MaxWidth(previewOuterW).MaxHeight(innerH+2).Render(previewInner)

	// Compose
	main := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, " ", previewPanel)
	left := " j/k:nav  enter:open  h:back  .:hidden  ctrl+d/u:scroll  q:quit"
	right := ""
	if len(m.visible) > 0 {
		right = fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.visible))
	}
	gap := m.width - len(left) - len(right)
	if gap < 1 {
		gap = 1
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
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
