# Tree-Glow Architecture

> A lean TUI file manager built with Go + Bubble Tea

## Table of Contents

1. [Overview](#overview)
2. [Design Philosophy](#design-philosophy)
3. [Components](#components)
4. [Data Flow](#data-flow)
5. [Rendering Pipeline](#rendering-pipeline)
6. [Configuration](#configuration)
7. [Performance](#performance)
8. [Future Plans](#future-plans)

---

## Overview

Tree-Glow is a terminal file manager that combines the best aspects of several tools:

- **glow** — beautiful markdown rendering with glamour
- **yazi** — fast file preview with syntax highlighting
- **warp** — clean, modern terminal aesthetic

The application uses a **25/75 split layout**: a collapsible file tree on the left, and a rich file preview on the right.

### Key Technologies

| Technology | Purpose | Version |
|-----------|---------|---------|
| Go | Core language | 1.22+ |
| Bubble Tea | TUI framework | v1.3.x |
| Lip Gloss | Styling/layout | v1.1.x |
| Glamour | Markdown rendering | v1.0.x |
| Chroma | Syntax highlighting | v2.23.x |

## Design Philosophy

### Simplicity First

> "Perfection is achieved not when there is nothing more to add, but when there is nothing left to take away." — Antoine de Saint-Exupéry

We follow three core principles:

1. **Minimal dependencies** — only what's needed
2. **Fast startup** — under 50ms cold start
3. **Intuitive controls** — vim-like navigation

### The Elm Architecture

Bubble Tea follows the [Elm Architecture](https://guide.elm-lang.org/architecture/), which means:

```
┌─────────┐     ┌────────┐     ┌──────┐
│  Model  │────▶│ Update │────▶│ View │
│ (state) │     │ (logic)│     │(HTML)│
└─────────┘     └────────┘     └──────┘
     ▲                              │
     │          ┌────────┐          │
     └──────────│  Msg   │◀─────────┘
                │(events)│
                └────────┘
```

Every interaction produces a **message**, which flows through **Update** to produce a new **Model**, which is rendered by **View**. This makes the application:

- Predictable
- Testable
- Easy to reason about

## Components

### File Tree (`tree.go`)

The file tree is a recursive data structure:

```go
type node struct {
    name     string
    path     string
    isDir    bool
    children []*node
    expanded bool
    depth    int
    parent   *node
}
```

#### Features

- **Lazy loading**: Children are loaded only when a directory is expanded
- **Sorted**: Directories first, then alphabetical (case-insensitive)
- **Filtered**: Hidden files toggled with `.` key
- **Icons**: File-type specific icons using Nerd Font glyphs

#### Supported File Icons

| Extension | Icon | Category |
|-----------|------|----------|
| `.go` | `` | Go source |
| `.md` | `` | Markdown |
| `.py` | `` | Python |
| `.js/.ts` | `` | JavaScript/TypeScript |
| `.rs` | `` | Rust |
| `.json` | `` | Data |
| `.yaml` | `` | Config |
| `.sh` | `` | Shell |
| `.png/.jpg` | `` | Image |
| `.lock` | `` | Lock file |

### Preview Engine (`preview.go`)

The preview engine handles multiple file types:

#### Markdown Files

Markdown files are rendered using **Glamour**, which provides:

- Full CommonMark support
- Syntax-highlighted code blocks
- Styled headings, lists, and tables
- Blockquote formatting
- Link rendering

Example of how a code block renders:

```python
def fibonacci(n: int) -> list[int]:
    """Generate Fibonacci sequence up to n terms."""
    if n <= 0:
        return []
    elif n == 1:
        return [0]
    
    sequence = [0, 1]
    while len(sequence) < n:
        next_val = sequence[-1] + sequence[-2]
        sequence.append(next_val)
    
    return sequence

# Generate first 10 Fibonacci numbers
result = fibonacci(10)
print(f"Fibonacci: {result}")
```

#### Code Files

Source code files are highlighted using **Chroma** with the Dracula theme:

```go
// renderCode highlights source code with line numbers
func renderCode(path, content string) string {
    lexer := lexers.Match(path)
    if lexer == nil {
        lexer = lexers.Fallback
    }
    
    style := styles.Get("dracula")
    formatter := formatters.Get("terminal256")
    
    iterator, _ := lexer.Tokenise(nil, content)
    var buf strings.Builder
    formatter.Format(&buf, style, iterator)
    
    return addLineNumbers(buf.String())
}
```

#### Binary Files

Binary files show a simple info card:

```
  Binary file (1,234,567 bytes)
```

#### Large Files

Files over 1MB show a size warning rather than attempting to render.

### Model (`model.go`)

The model holds all application state:

```go
type model struct {
    root          *node       // file tree root
    visible       []*node     // flattened visible nodes
    cursor        int         // selected index
    width, height int         // terminal dimensions
    treeScroll    int         // tree viewport offset
    previewScroll int         // preview viewport offset
    showHidden    bool        // show dotfiles
    cachedPath    string      // cached preview file
    cachedLines   []string    // cached rendered lines
}
```

## Data Flow

### Navigation Flow

```
User presses 'j'
    │
    ▼
KeyMsg("j") ─── Update() ───▶ cursor++
    │                           previewScroll = 0
    │                           refreshPreviewCache()
    ▼
View() ─── renderTree() ───▶ highlight new cursor
    │   └── buildPreviewLines() ───▶ show new file
    ▼
Terminal renders new frame
```

### Expand/Collapse Flow

```
User presses 'enter' on directory
    │
    ▼
KeyMsg("enter") ─── Update() ───▶ node.toggle()
    │                               flatten(root)
    │                               recalculate visible[]
    ▼
View() ─── renderTree() ───▶ show expanded tree
    │   └── preview unchanged (same cursor)
    ▼
Terminal renders new frame
```

## Rendering Pipeline

### Frame Budget

At 60fps, each frame has ~16ms to:

1. Process messages (~1ms)
2. Update model (~1ms)  
3. Render view (~5-10ms)
4. Terminal output (~2-5ms)

### Preview Caching Strategy

```
File selected ──▶ Check cache ──▶ HIT ──▶ Use cached lines
                       │
                       ▼
                      MISS
                       │
                       ▼
                  Render preview
                       │
                  ┌────┴─────┐
                  │          │
               Markdown    Code
                  │          │
               glamour    chroma
                  │          │
                  └────┬─────┘
                       │
                       ▼
                  Split into lines
                       │
                       ▼
                  Store in cache
```

**Why cache?**

- Glamour rendering: ~50-200ms per file
- Chroma highlighting: ~10-50ms per file
- View() is called 60x/sec
- Without cache: 3-12 seconds of CPU per second 💀

### Layout Math

```
Terminal: width × height
├── Tree panel (25% width)
│   ├── Border: 2 cols × 2 rows
│   └── Content: (25%w - 2) × (h - 3)
├── Gap: 1 col
├── Preview panel (75% width - 2)
│   ├── Title bar: 1 row
│   ├── Border: 2 cols × 2 rows  
│   └── Content: (75%w - 4) × (h - 4)
└── Status bar: 1 row
```

## Configuration

### Planned Configuration File

```toml
[layout]
tree_width = 0.25        # ratio of screen width
show_hidden = false       # show dotfiles by default

[preview]
max_file_size = "1MB"     # max file to preview
theme = "dracula"         # chroma theme
word_wrap = true          # wrap long lines

[keys]
navigate_up = "k"
navigate_down = "j"
expand = "enter"
collapse = "h"
toggle_hidden = "."
scroll_down = "ctrl+d"
scroll_up = "ctrl+u"
quit = "q"
```

## Performance

### Benchmarks (Target)

| Operation | Time | Notes |
|-----------|------|-------|
| Startup | <50ms | Lazy tree loading |
| Directory expand | <10ms | Single readdir |
| Markdown preview | <200ms | Cached after first |
| Code highlight | <50ms | Cached after first |
| Frame render | <10ms | Pre-clipped content |
| Scroll | <1ms | Array slice |

### Memory Usage

- Base: ~5MB
- Per cached file: ~10KB average
- Tree nodes: ~200 bytes each
- Typical session: <20MB

## Future Plans

### Phase 1: Core Features ✅

- [x] File tree navigation
- [x] Syntax-highlighted preview
- [x] Markdown rendering
- [x] Mouse support
- [x] Scroll support

### Phase 2: Edit Mode 🚧

- [ ] Toggle preview → edit mode
- [ ] Basic text editing
- [ ] Save with confirmation
- [ ] Syntax highlighting in edit mode

### Phase 3: Advanced Features 📋

- [ ] File search (fuzzy finder)
- [ ] Bookmarks
- [ ] File operations (copy, move, delete)
- [ ] Git status integration
- [ ] Multiple tabs
- [ ] Split view
- [ ] Image preview (sixel/kitty)
- [ ] Custom themes

### Phase 4: Plugin System 🔮

- [ ] Lua scripting support
- [ ] Custom preview renderers
- [ ] Custom key bindings
- [ ] Event hooks

---

## Contributing

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `golangci-lint`
- Keep functions under 50 lines
- Prefer clarity over cleverness

### Testing

```bash
go test ./...                    # unit tests
go test -race ./...              # race detection
go test -bench=. ./...           # benchmarks
```

### Architecture Decision Records

| ADR | Decision | Status |
|-----|----------|--------|
| 001 | Use Bubble Tea over tcell | Accepted |
| 002 | Cache preview per file | Accepted |
| 003 | Manual layout over lipgloss | Accepted |
| 004 | Glamour for markdown | Accepted |
| 005 | Dracula as default theme | Proposed |

---

*This document is maintained as part of the tree-glow project.*
*Last updated: 2024-01-15*
