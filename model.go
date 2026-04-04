package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	gif   *animatedGif     // non-nil for half-block animated GIFs
	kitty *kittyImageState // non-nil for kitty static images
}

// gifTickMsg triggers the next GIF frame.
type gifTickMsg struct{}

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

	// animated gif
	activeGif *animatedGif

	// kitty image
	kittyImg *kittyImageState

	// edit mode
	editMode bool
	editor   *editor
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
	m.activeGif = nil
	m.kittyImg = nil

	_, previewOuterW := m.layoutWidths()
	pw := previewOuterW - 6
	if pw < 20 {
		pw = 20
	}
	ph := m.height - 6
	if ph < 5 {
		ph = 5
	}

	return func() tea.Msg {
		ext := strings.ToLower(filepath.Ext(path))
		isMd := ext == ".md" || ext == ".markdown"

		// Animated GIF — always use half-blocks (kitty clear+redraw flickers)
		if isGifFile(ext) {
			ag := loadAnimatedGif(path, pw, ph)
			if ag != nil {
				lines := strings.Split(ag.CurrentFrame(), "\n")
				return previewMsg{path: path, lines: lines, gif: ag}
			}
		}

		// Static images — try kitty protocol for pixel-perfect rendering
		if isImageFile(ext) && !isGifFile(ext) && isKittyAvailable() {
			ki, err := newKittyImage(path, pw, ph)
			if err == nil {
				// cachedLines just holds the header + blank space
				// actual image is overlaid via escape sequences in View
				header := ki.Header()
				lines := []string{header, ""}
				// Fill with blank lines so scroll math works
				for i := 0; i < ph; i++ {
					lines = append(lines, "")
				}
				return previewMsg{path: path, lines: lines, kitty: ki}
			}
		}

		// Read raw file content for clipboard
		var rawLines []string
		if data, err := os.ReadFile(path); err == nil {
			rawLines = strings.Split(string(data), "\n")
		}

		// Render for display
		result := renderPreview(path, pw, ph)
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
			m.activeGif = msg.gif
			m.kittyImg = msg.kitty
			if msg.gif != nil && len(msg.gif.frames) > 1 {
				// Start animation tick
				delay := msg.gif.delays[0]
				return m, tea.Tick(delay, func(time.Time) tea.Msg { return gifTickMsg{} })
			}
		}
		return m, nil

	case gifTickMsg:
		if m.activeGif != nil && len(m.activeGif.frames) > 1 {
			delay := m.activeGif.Advance()
			m.cachedLines = strings.Split(m.activeGif.CurrentFrame(), "\n")
			return m, tea.Tick(delay, func(time.Time) tea.Msg { return gifTickMsg{} })
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
				// Capture values for goroutine to avoid data race
				startY, endY := m.selectionRange()
				isMd := m.cachedIsMarkdown
				cachedLines := m.cachedLines
				rawLines := m.cachedRawLines
				go copyToClipboard(startY, endY, isMd, cachedLines, rawLines)
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
	clipped := hardClip(full, m.width, m.height)

	// Kitty image overlay — appended AFTER hardClip to avoid mangling.
	// Only clear when no kitty content is active (navigated away).
	// Animation frames overwrite in-place without clearing.
	if useKitty {
		treeOuterW, _ := m.layoutWidths()
		imgRow := 3
		imgCol := treeOuterW + 3
		overlay := ""
		if m.kittyImg != nil {
			overlay = m.kittyImg.OverlayString(imgRow, imgCol)
		}
		if overlay != "" {
			clipped += overlay
		} else {
			clipped += kittyClearImages()
		}
	}

	return clipped
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
// Safe to call from a goroutine — all data passed by value.
func copyToClipboard(startY, endY int, isMd bool, cachedLines, rawLines []string) {
	if startY > endY {
		startY, endY = endY, startY
	}
	if startY < 0 {
		startY = 0
	}

	var text string
	if isMd || len(rawLines) == 0 {
		if endY >= len(cachedLines) {
			endY = len(cachedLines) - 1
		}
		if startY > endY {
			return
		}
		var lines []string
		for i := startY; i <= endY; i++ {
			lines = append(lines, ansi.Strip(cachedLines[i]))
		}
		text = strings.Join(lines, "\n")
	} else {
		if endY >= len(rawLines) {
			endY = len(rawLines) - 1
		}
		if startY > endY {
			return
		}
		text = strings.Join(rawLines[startY:endY+1], "\n")
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
	_, previewOuterW := m.layoutWidths()
	editW := previewOuterW - 6
	editH := m.height - 5
	if editW < 20 {
		editW = 20
	}
	if editH < 3 {
		editH = 3
	}

	ed, err := newEditor(path, editW, editH)
	if err != nil {
		return nil
	}
	m.editMode = true
	m.editor = ed
	return nil
}

func (m model) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.editor != nil {
			_, previewOuterW := m.layoutWidths()
			m.editor.width = previewOuterW - 6
			m.editor.height = m.height - 5
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if m.editor != nil {
				if err := m.editor.Save(); err != nil {
					// Stay in edit mode on error
					return m, nil
				}
			}
			m.editMode = false
			m.editor = nil
			m.cachedPath = ""
			m.cachedLines = nil
			m.cachedRawLines = nil
			cmd := m.requestPreview()
			return m, cmd
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.editMode = false
			m.editor = nil
			m.cachedPath = ""
			m.cachedLines = nil
			m.cachedRawLines = nil
			cmd := m.requestPreview()
			return m, cmd
		default:
			if m.editor != nil {
				m.editor.Update(msg)
			}
			return m, nil
		}

	case tea.MouseMsg:
		if m.editor == nil {
			return m, nil
		}
		treeW, _ := m.layoutWidths()
		switch {
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress && msg.X >= treeW+2:
			lineIdx := msg.Y - 3 + m.editor.scrollY // 3 = header + border top + title inside border
			if lineIdx >= 0 && lineIdx < len(m.editor.lines) {
				m.editor.cursorY = lineIdx
				numW := len(fmt.Sprintf("%d", len(m.editor.lines)))
				contentX := msg.X - treeW - 2 - numW - 2
				if contentX < 0 {
					contentX = 0
				}
				lineLen := len([]rune(m.editor.lines[lineIdx]))
				if contentX > lineLen {
					contentX = lineLen
				}
				m.editor.cursorX = contentX
			}
		case msg.Button == tea.MouseButtonWheelUp && msg.X >= treeW:
			m.editor.scrollY -= 3
			if m.editor.scrollY < 0 {
				m.editor.scrollY = 0
			}
		case msg.Button == tea.MouseButtonWheelDown && msg.X >= treeW:
			m.editor.scrollY += 3
			maxScroll := len(m.editor.lines) - m.editor.height
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.editor.scrollY > maxScroll {
				m.editor.scrollY = maxScroll
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) viewEditMode() string {
	treeOuterW, previewOuterW := m.layoutWidths()
	treeInnerW := treeOuterW - 2
	innerH := m.height - 3
	if innerH < 1 {
		innerH = 1
	}

	// Tree
	treeContent := buildExactLines(m.buildTreeLines(treeInnerW, innerH), innerH, treeInnerW)
	treePanel := borderStyle.Width(treeInnerW).Height(innerH).MaxWidth(treeOuterW).MaxHeight(innerH+2).Render(treeContent)

	// Edit panel
	previewInnerW := previewOuterW - 2

	// Title bar
	indicator := "EDIT"
	titleSt := editTitleClean
	if m.editor != nil && m.editor.dirty {
		indicator = "EDIT *"
		titleSt = editTitleDirty
	}
	editPath := ""
	if m.editor != nil {
		editPath = m.editor.path
	}
	titleText := indicator + "  " + truncate(editPath, previewInnerW-len(indicator)-4)
	header := titleSt.Width(previewOuterW).Render(titleText)

	// Editor content with syntax highlighting
	editContent := ""
	if m.editor != nil {
		editContent = m.editor.Render(previewInnerW)
	}
	editPanel := borderStyle.Width(previewInnerW).Height(innerH-1).MaxWidth(previewOuterW).MaxHeight(innerH+1).Render(editContent)

	previewPanel := lipgloss.JoinVertical(lipgloss.Left, header, editPanel)
	main := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, " ", previewPanel)

	// Status bar
	left := " ctrl+s:save  esc:cancel  ctrl+c:quit"
	right := ""
	if m.editor != nil && m.editor.dirty {
		right = " modified "
	}
	if m.editor != nil {
		right = fmt.Sprintf(" Ln %d, Col %d %s", m.editor.cursorY+1, m.editor.cursorX+1, right)
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

