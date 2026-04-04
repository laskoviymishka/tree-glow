package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const maxFileSize = 1 << 20 // 1MB

var numStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// renderPreview returns the full rendered content for a file.
// Called once per file selection, cached by the model.
func renderPreview(path string, width int) string {
	info, err := os.Stat(path)
	if err != nil {
		return dimStyle.Render("  Cannot read: " + err.Error())
	}

	if info.IsDir() {
		return renderDirPreview(path)
	}

	if !info.Mode().IsRegular() {
		return dimStyle.Render("  Not a regular file")
	}

	if info.Size() > maxFileSize {
		return dimStyle.Render(fmt.Sprintf("  File too large (%d MB)", info.Size()/(1<<20)))
	}

	if info.Size() == 0 {
		return dimStyle.Render("  (empty file)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return dimStyle.Render("  Cannot read: " + err.Error())
	}

	if !utf8.Valid(data) {
		return dimStyle.Render(fmt.Sprintf("  Binary file (%d bytes)", len(data)))
	}

	content := string(data)
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".md" || ext == ".markdown" {
		return renderMarkdown(content, width)
	}

	return renderCode(path, content)
}

func renderMarkdown(content string, width int) string {
	if width < 10 {
		width = 10
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return out
}

func renderCode(path, content string) string {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return addLineNumbers(content)
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return addLineNumbers(content)
	}

	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return addLineNumbers(content)
	}

	return addLineNumbers(buf.String())
}

func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	numWidth := len(fmt.Sprintf("%d", len(lines)))

	var buf strings.Builder
	for i, line := range lines {
		num := fmt.Sprintf("%*d", numWidth, i+1)
		buf.WriteString(numStyle.Render(num))
		buf.WriteString("  ")
		buf.WriteString(line)
		if i < len(lines)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func renderDirPreview(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return dimStyle.Render("  Cannot read directory")
	}

	var buf strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	buf.WriteString(headerStyle.Render(fmt.Sprintf("  %s/", filepath.Base(path))))
	buf.WriteString("\n\n")

	dirs, files := 0, 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			dirs++
			buf.WriteString(fmt.Sprintf("   %s/\n", e.Name()))
		} else {
			files++
			info, _ := e.Info()
			size := ""
			if info != nil {
				size = formatSize(info.Size())
			}
			buf.WriteString(fmt.Sprintf("   %s  %s\n", e.Name(), dimStyle.Render(size)))
		}
	}

	buf.WriteString(fmt.Sprintf("\n  %s", dimStyle.Render(fmt.Sprintf("%d dirs, %d files", dirs, files))))
	return buf.String()
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
