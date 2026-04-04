# tree-glow

A lean TUI file manager with syntax-highlighted preview and inline editing.

Built with Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Glamour](https://github.com/charmbracelet/glamour) + [Chroma](https://github.com/alecthomas/chroma).

![demo](demo.gif)

## Install

```bash
go install github.com/laskoviymishka/tree-glow@latest
```

Or build from source:

```bash
git clone https://github.com/laskoviymishka/tree-glow.git
cd tree-glow
go build -o tree-glow .
```

## Usage

```bash
./tree-glow          # current directory
./tree-glow ~/code   # specific path
```

## Features

**File tree** (25% width)
- Collapsible directories with file-type icons
- Keyboard and mouse navigation
- Hidden files toggle

**Preview** (75% width)
- Syntax-highlighted code (Chroma, Dracula theme)
- Rendered markdown (Glamour)
- Directory listings with file sizes
- Async rendering — UI stays responsive on large files

**Inline editor**
- Press `e` to edit any file in-place
- Syntax highlighting while editing
- Word navigation (opt+arrows on macOS)
- Save with `ctrl+s`, cancel with `esc`
- Preserves file permissions

**Text selection**
- Click and drag in preview to select lines
- Auto-copies raw text to clipboard on release
- Code files: copies source without line numbers
- Markdown files: copies rendered text

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `enter` / `l` | Expand directory |
| `h` | Collapse / go to parent |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `.` | Toggle hidden files |
| `q` | Quit |

### Preview

| Key | Action |
|-----|--------|
| `ctrl+d` | Scroll preview down |
| `ctrl+u` | Scroll preview up |
| Mouse wheel | Scroll tree (left) or preview (right) |
| Click + drag | Select text in preview |

### Editor

| Key | Action |
|-----|--------|
| `e` | Enter edit mode |
| `ctrl+s` | Save and exit |
| `esc` | Cancel and exit |
| `opt+←` / `opt+→` | Word left / right |
| `opt+backspace` | Delete word left |
| `opt+d` | Delete word right |
| `ctrl+a` / `ctrl+e` | Home / End |
| `ctrl+k` | Kill to end of line |
| Click | Place cursor |

### Ghostty users

To enable `cmd+s` for save, add to `~/.config/ghostty/config`:

```
keybind = super+s=text:\x13
```

## Architecture

```
tree-glow/
├── main.go      # entry point
├── tree.go      # file tree data model
├── preview.go   # syntax highlighting + markdown rendering
├── editor.go    # custom editor with chroma highlighting
└── model.go     # TUI model, view, layout, input handling
```

## License

MIT
