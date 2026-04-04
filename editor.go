package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("230")).
			Foreground(lipgloss.Color("0"))

	editTitleDirty = lipgloss.NewStyle().
			Background(lipgloss.Color("166")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	editTitleClean = lipgloss.NewStyle().
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)
)

type editor struct {
	lines    []string
	cursorX  int // column (rune index)
	cursorY  int // row (line index)
	scrollY  int // viewport scroll offset
	path     string
	dirty    bool
	width    int
	height   int
	fileMode fs.FileMode // original file permissions
}

func newEditor(path string, width, height int) (*editor, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return &editor{
		lines:    lines,
		path:     path,
		width:    width,
		height:   height,
		fileMode: info.Mode().Perm(),
	}, nil
}

func (e *editor) Update(msg tea.KeyMsg) {
	switch msg.String() {
	case "up":
		if e.cursorY > 0 {
			e.cursorY--
			e.clampX()
		}
	case "down":
		if e.cursorY < len(e.lines)-1 {
			e.cursorY++
			e.clampX()
		}
	case "left":
		if e.cursorX > 0 {
			e.cursorX--
		} else if e.cursorY > 0 {
			e.cursorY--
			e.cursorX = len([]rune(e.lines[e.cursorY]))
		}
	case "right":
		runes := []rune(e.lines[e.cursorY])
		if e.cursorX < len(runes) {
			e.cursorX++
		} else if e.cursorY < len(e.lines)-1 {
			e.cursorY++
			e.cursorX = 0
		}
	case "alt+left", "ctrl+left", "alt+b":
		e.cursorX = e.wordLeft()
	case "alt+right", "ctrl+right", "alt+f":
		e.cursorX = e.wordRight()
	case "alt+up", "ctrl+up":
		e.cursorY -= 5
		if e.cursorY < 0 {
			e.cursorY = 0
		}
		e.clampX()
	case "alt+down", "ctrl+down":
		e.cursorY += 5
		if e.cursorY >= len(e.lines) {
			e.cursorY = len(e.lines) - 1
		}
		e.clampX()
	case "home", "ctrl+a":
		e.cursorX = 0
	case "end", "ctrl+e":
		e.cursorX = len([]rune(e.lines[e.cursorY]))
	case "alt+backspace", "ctrl+backspace", "ctrl+w":
		e.deleteWordLeft()
		e.dirty = true
	case "alt+d":
		e.deleteWordRight()
		e.dirty = true
	case "ctrl+k":
		// Kill to end of line
		runes := []rune(e.lines[e.cursorY])
		if e.cursorX < len(runes) {
			e.lines[e.cursorY] = string(runes[:e.cursorX])
			e.dirty = true
		}
	case "enter":
		e.insertNewline()
		e.dirty = true
	case "backspace":
		e.backspace()
		e.dirty = true
	case "delete":
		e.deleteChar()
		e.dirty = true
	case "tab":
		e.insertText("\t")
		e.dirty = true
	default:
		// Insert printable characters
		if len(msg.Runes) > 0 {
			e.insertText(string(msg.Runes))
			e.dirty = true
		}
	}

	// Keep cursor in viewport
	if e.cursorY < e.scrollY {
		e.scrollY = e.cursorY
	}
	if e.cursorY >= e.scrollY+e.height {
		e.scrollY = e.cursorY - e.height + 1
	}
	// Clamp scrollY after deletions may have shrunk the file
	maxScroll := len(e.lines) - e.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scrollY > maxScroll {
		e.scrollY = maxScroll
	}
}

func (e *editor) insertText(s string) {
	runes := []rune(e.lines[e.cursorY])
	if e.cursorX > len(runes) {
		e.cursorX = len(runes)
	}
	newRunes := make([]rune, 0, len(runes)+len([]rune(s)))
	newRunes = append(newRunes, runes[:e.cursorX]...)
	newRunes = append(newRunes, []rune(s)...)
	newRunes = append(newRunes, runes[e.cursorX:]...)
	e.lines[e.cursorY] = string(newRunes)
	e.cursorX += len([]rune(s))
}

func (e *editor) insertNewline() {
	runes := []rune(e.lines[e.cursorY])
	if e.cursorX > len(runes) {
		e.cursorX = len(runes)
	}
	before := string(runes[:e.cursorX])
	after := string(runes[e.cursorX:])

	// Insert new line
	newLines := make([]string, 0, len(e.lines)+1)
	newLines = append(newLines, e.lines[:e.cursorY]...)
	newLines = append(newLines, before)
	newLines = append(newLines, after)
	if e.cursorY+1 < len(e.lines) {
		newLines = append(newLines, e.lines[e.cursorY+1:]...)
	}
	e.lines = newLines
	e.cursorY++
	e.cursorX = 0
}

func (e *editor) backspace() {
	if e.cursorX > 0 {
		runes := []rune(e.lines[e.cursorY])
		if e.cursorX > len(runes) {
			e.cursorX = len(runes)
		}
		e.lines[e.cursorY] = string(runes[:e.cursorX-1]) + string(runes[e.cursorX:])
		e.cursorX--
	} else if e.cursorY > 0 {
		// Merge with previous line
		prevLen := len([]rune(e.lines[e.cursorY-1]))
		e.lines[e.cursorY-1] += e.lines[e.cursorY]
		e.lines = append(e.lines[:e.cursorY], e.lines[e.cursorY+1:]...)
		e.cursorY--
		e.cursorX = prevLen
	}
}

func (e *editor) deleteChar() {
	runes := []rune(e.lines[e.cursorY])
	if e.cursorX < len(runes) {
		e.lines[e.cursorY] = string(runes[:e.cursorX]) + string(runes[e.cursorX+1:])
	} else if e.cursorY < len(e.lines)-1 {
		// Merge with next line
		e.lines[e.cursorY] += e.lines[e.cursorY+1]
		e.lines = append(e.lines[:e.cursorY+1], e.lines[e.cursorY+2:]...)
	}
}

func (e *editor) wordLeft() int {
	runes := []rune(e.lines[e.cursorY])
	x := e.cursorX
	if x > len(runes) {
		x = len(runes)
	}
	// Skip spaces backwards
	for x > 0 && isSpace(runes[x-1]) {
		x--
	}
	// Skip word chars backwards
	for x > 0 && !isSpace(runes[x-1]) {
		x--
	}
	return x
}

func (e *editor) wordRight() int {
	runes := []rune(e.lines[e.cursorY])
	x := e.cursorX
	// Skip word chars forward
	for x < len(runes) && !isSpace(runes[x]) {
		x++
	}
	// Skip spaces forward
	for x < len(runes) && isSpace(runes[x]) {
		x++
	}
	return x
}

func (e *editor) deleteWordLeft() {
	target := e.wordLeft()
	if target == e.cursorX {
		// Nothing to delete on this line, try backspace
		e.backspace()
		return
	}
	runes := []rune(e.lines[e.cursorY])
	e.lines[e.cursorY] = string(runes[:target]) + string(runes[e.cursorX:])
	e.cursorX = target
}

func (e *editor) deleteWordRight() {
	target := e.wordRight()
	runes := []rune(e.lines[e.cursorY])
	if target == e.cursorX && e.cursorY < len(e.lines)-1 {
		// At end of line, merge with next
		e.lines[e.cursorY] += e.lines[e.cursorY+1]
		e.lines = append(e.lines[:e.cursorY+1], e.lines[e.cursorY+2:]...)
		return
	}
	e.lines[e.cursorY] = string(runes[:e.cursorX]) + string(runes[target:])
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func (e *editor) clampX() {
	maxX := len([]rune(e.lines[e.cursorY]))
	if e.cursorX > maxX {
		e.cursorX = maxX
	}
}

func (e *editor) Save() error {
	content := strings.Join(e.lines, "\n")
	err := os.WriteFile(e.path, []byte(content), e.fileMode)
	if err == nil {
		e.dirty = false
	}
	return err
}

func (e *editor) Value() string {
	return strings.Join(e.lines, "\n")
}

// Render returns syntax-highlighted content with cursor for the visible viewport.
func (e *editor) Render(width int) string {
	// Highlight the full visible range
	content := e.Value()
	highlighted := renderCode(e.path, content)
	hLines := strings.Split(highlighted, "\n")

	numWidth := len(fmt.Sprintf("%d", len(e.lines)))
	var out strings.Builder

	end := e.scrollY + e.height
	if end > len(hLines) {
		end = len(hLines)
	}
	if end > len(e.lines) {
		end = len(e.lines)
	}

	for i := e.scrollY; i < end; i++ {
		line := ""
		if i < len(hLines) {
			line = hLines[i]
		}

		if i == e.cursorY {
			// Render cursor on this line
			line = e.renderCursorLine(i, numWidth, width)
		} else {
			line = ansi.Truncate(line, width, "")
		}

		out.WriteString(line)
		if i < end-1 {
			out.WriteString("\n")
		}
	}

	// Pad remaining lines
	for i := end; i < e.scrollY+e.height; i++ {
		out.WriteString("\n")
	}

	return out.String()
}

// renderCursorLine renders the cursor line with the cursor character highlighted.
func (e *editor) renderCursorLine(lineIdx, numWidth, width int) string {
	rawLine := e.lines[lineIdx]
	runes := []rune(rawLine)

	// Build: line_number + "  " + before_cursor + cursor_char + after_cursor
	num := fmt.Sprintf("%*d", numWidth, lineIdx+1)
	numRendered := numStyle.Render(num) + "  "

	curX := e.cursorX
	if curX > len(runes) {
		curX = len(runes)
	}

	before := string(runes[:curX])
	cursorChar := " " // cursor at end of line
	after := ""
	if curX < len(runes) {
		cursorChar = string(runes[curX])
		after = string(runes[curX+1:])
	}

	result := numRendered + before + cursorStyle.Render(cursorChar) + after
	return ansi.Truncate(result, width, "")
}
