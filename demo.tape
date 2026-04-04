Output demo.gif

Set Shell zsh
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 10
Set TypingSpeed 50ms

# Start directly in the app
Hide
Type "cd /Users/andrei.tserakhau/tree-glow && ./tree-glow demo/"
Enter
Sleep 2s
Show

Sleep 1s

# Navigate down the tree
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s

# Expand config/
Type "l"
Sleep 0.6s
Type "j"
Sleep 0.5s

# Preview YAML
Sleep 1.2s

# Navigate to docs
Type "h"
Sleep 0.3s
Type "j"
Sleep 0.3s

# Expand docs/
Type "l"
Sleep 0.5s
Type "j"
Sleep 0.5s

# Preview markdown (guide.md)
Sleep 2s

# Scroll preview
Ctrl+d
Sleep 1s
Ctrl+u
Sleep 1s

# Navigate to src
Type "h"
Sleep 0.3s
Type "j"
Sleep 0.3s

# Expand src/
Type "l"
Sleep 0.3s

# Expand components/
Type "j"
Sleep 0.3s
Type "l"
Sleep 0.3s
Type "j"
Sleep 0.5s

# Preview TSX file
Sleep 1.5s

# Navigate to Python file
Type "h"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "l"
Sleep 0.3s
Type "j"
Sleep 0.5s

# Preview Python
Sleep 1.5s

# Enter edit mode
Type "e"
Sleep 1.5s

# Type a comment
Type "# edited in tree-glow!"
Enter
Sleep 1s

# Exit edit mode
Escape
Sleep 1s

# Go to top
Type "g"
Sleep 0.5s

# Preview README.md
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 1.5s

# Scroll the markdown
Ctrl+d
Sleep 1s
Ctrl+u
Sleep 1s

# Quit
Type "q"
Sleep 0.3s
