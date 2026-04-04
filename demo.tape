Output demo.gif

Set Shell zsh
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 10

Type "cd /Users/andrei.tserakhau/tree-glow && ./tree-glow demo/"
Enter
Sleep 1.5s

# Navigate the tree
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.4s

# Expand a directory
Type "l"
Sleep 0.8s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.4s

# Preview a Go file
Type "j"
Sleep 1.5s

# Scroll the preview
Ctrl+d
Sleep 1s
Ctrl+u
Sleep 1s

# Navigate to markdown
Type "g"
Sleep 0.5s
Type "j"
Sleep 0.5s
# Go to README.md
Type "l"
Sleep 0.3s
Type "j"
Sleep 1.5s

# Scroll markdown
Ctrl+d
Sleep 1s
Ctrl+u
Sleep 1s

# Navigate to a code file and enter edit mode
Type "g"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "l"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 1s

# Enter edit mode
Type "e"
Sleep 1.5s

# Type something
Type "// edited with tree-glow!"
Sleep 1s

# Exit without saving
Escape
Sleep 1s

# Quit
Type "q"
Sleep 0.5s
