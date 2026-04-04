# User Guide

## Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `enter` / `l` | Expand directory |
| `h` | Collapse / go to parent |
| `.` | Toggle hidden files |
| `ctrl+d` | Scroll preview down |
| `ctrl+u` | Scroll preview up |

## Preview

Files are previewed based on type:

- **Markdown** files render with full styling
- **Code** files get syntax highlighting
- **Directories** show a summary listing
- **Binary** files show size info

## Architecture

```
tree-glow/
├── main.go      # entry point
├── tree.go      # file tree data model
├── preview.go   # file rendering
└── model.go     # TUI model + view
```
