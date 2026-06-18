# movelooper demo — recorded with VHS (https://github.com/charmbracelet/vhs)
#
# Prerequisites:
#   go install github.com/charmbracelet/vhs@latest
#   $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o movelooper .
#
# Record (Docker):
#   docker run --rm -v "${PWD}:/vhs" ghcr.io/charmbracelet/vhs demo.tape

Output demo.gif

Set Shell         "bash"
Set FontSize      13
Set Width         1600
Set Height        900
Set Theme         "Catppuccin Mocha"
Set Framerate     30
Set PlaybackSpeed 1.0

# -- Open the editor ----------------------------------------------------------

Type "./movelooper edit --theme banana --no-save-confirm --no-delete-confirm --config ./movelooper.yaml"
Sleep 200ms
Enter
Sleep 2.5s

# -- Main list: browse with hint panel ----------------------------------------

# Hint on "configuration" (first item, already selected)
Type "h"
Sleep 1.8s

# Move to "categories" and show its hint
Down
Sleep 500ms
Type "h"
Sleep 1.8s

# -- Configuration block (KindObject -- struct tree) --------------------------

Up
Sleep 300ms
Enter
Sleep 1.5s

# Navigate the sub-fields: output → log-file → log-level → show-caller
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms

# Switch to the YAML preview (right panel) to show live rendering
Tab
Sleep 1.5s

# Back to the tree
Tab
Sleep 500ms

# Commit and return to main list
Ctrl+S
Sleep 800ms

# -- Categories collection (KindList) -----------------------------------------

Down
Sleep 400ms
Enter
Sleep 1.5s

# Scroll through the list entries: images → audio → video → documents
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 350ms

# -- Category entry block editor (images) -- open and drill into source -------

Up
Up
Up
Up
Sleep 500ms
Enter
Sleep 1.5s

# Navigate to the "source" field (2nd row in typical order: name, source, ...)
Down
Sleep 400ms

# Drill into "source" nested editor with Right arrow
Right
Sleep 1.8s

# Inside source: navigate path → extensions → filter
Down
Sleep 400ms
Down
Sleep 400ms

# Switch to YAML editor to show the full nested block
Tab
Sleep 1.8s

# Back to the nested tree
Tab
Sleep 400ms

# Back to the parent category block editor
Escape
Sleep 800ms

# -- Add a new category entry -------------------------------------------------

# Back to the collection navigator
Escape
Sleep 600ms

# Jump to the bottom with G to land on [+ add new]
Type "G"
Sleep 500ms

# Create the new entry
Enter
Sleep 1.2s

# Open the freshly created entry
Enter
Sleep 1.5s

# Tab straight to the YAML editor to type a minimal category stub
Tab
Sleep 800ms

Type "name: demo-category"
Sleep 400ms
Enter
Type "enabled: false"
Sleep 600ms

# Commit the new entry
Ctrl+S
Sleep 800ms

# Back to collection navigator: new entry is visible
Escape
Sleep 800ms

# -- Delete the new entry and undo --------------------------------------------

# New entry should be selected (last one); delete it instantly (no-delete-confirm)
Ctrl+D
Sleep 1.2s

# Undo: entry comes back
Ctrl+U
Sleep 1.2s

# -- Back to main list --------------------------------------------------------

Escape
Sleep 600ms

# -- Validate the document ----------------------------------------------------

Ctrl+L
Sleep 2.5s

Escape
Sleep 500ms

# -- Quit ---------------------------------------------------------------------

Type "q"
Sleep 1.2s
