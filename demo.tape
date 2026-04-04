Output demo.gif

Set Shell zsh
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 10
Set TypingSpeed 30ms

# Start app hidden, wait extra for full render
Hide
Type "cd /Users/andrei.tserakhau/tree-glow && ./tree-glow demo/"
Enter
Sleep 3s
Show

Sleep 0.5s

# Navigate down the tree
Type "j"
Sleep 0.2s
Type "j"
Sleep 0.2s
Type "j"
Sleep 0.2s

# Expand config/
Type "l"
Sleep 0.3s
Type "j"
Sleep 0.3s

# Preview YAML
Sleep 0.8s

# Navigate to docs
Type "h"
Sleep 0.2s
Type "j"
Sleep 0.2s

# Expand docs/
Type "l"
Sleep 0.3s
Type "j"
Sleep 0.3s

# Preview markdown (guide.md)
Sleep 1s

# Scroll preview
Ctrl+d
Sleep 0.5s
Ctrl+u
Sleep 0.5s

# Navigate to src
Type "h"
Sleep 0.2s
Type "j"
Sleep 0.2s

# Expand src/
Type "l"
Sleep 0.2s

# Expand components/
Type "j"
Sleep 0.2s
Type "l"
Sleep 0.2s
Type "j"
Sleep 0.3s

# Preview TSX file
Sleep 0.8s

# Navigate to Python file
Type "h"
Sleep 0.2s
Type "j"
Sleep 0.2s
Type "l"
Sleep 0.2s
Type "j"
Sleep 0.3s

# Preview Python
Sleep 0.8s

# Enter edit mode
Type "e"
Sleep 0.8s

# Type a comment
Type "# edited in tree-glow!"
Enter
Sleep 0.5s

# Exit edit mode
Escape
Sleep 0.5s

# Go to top, find README
Type "g"
Sleep 0.2s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.15s
Type "j"
Sleep 0.8s

# Scroll the markdown
Ctrl+d
Sleep 0.5s
Ctrl+u
Sleep 0.5s

# Quit
Type "q"
Sleep 0.2s
