# movelooper edit: the config editor that knows the schema.
#
# Opens edit.yaml, walks to a category's destination, reads the hint for
# action (its type, default and allowed values), picks copy from the list the
# schema gives, and saves: the editor validates and shows the diff first.
#
# Run from this directory, with movelooper on your PATH:
#   vhs edit.tape
#
# The tape edits a copy in ./sandbox, so edit.yaml never changes.

Output edit.gif

Require ttyd
Require ffmpeg

Set Shell         "powershell"
Set FontSize      14
Set Width         1100
Set Height        640
Set Theme         "Catppuccin Mocha"
Set Framerate     30
Set PlaybackSpeed 1.0
Set TypingSpeed   35ms
Set Padding       16

# -- A copy to edit -------------------------------------------------------------

Hide
Type "Remove-Item -Recurse -Force sandbox -ErrorAction SilentlyContinue"
Enter
Type "New-Item -ItemType Directory -Force sandbox | Out-Null"
Enter
Type "Copy-Item edit.yaml sandbox\movelooper.yaml"
Enter
Type "cd sandbox"
Enter
Type "clear"
Enter
Show

# -- Open the config ------------------------------------------------------------

Sleep 500ms
Type "movelooper edit --config movelooper.yaml --theme default"
Sleep 500ms
Enter
Sleep 2.5s

# The block list: categories is in the file, the hint describes it.
Down
Sleep 1.5s

# Open categories and the images entry, then its destination.
Enter
Sleep 1s
Right
Sleep 800ms
Down 4
Sleep 500ms
Right
Sleep 1s

# Walk down the destination's fields; each one's hint follows the cursor.
Down
Sleep 900ms
Down
Sleep 900ms
Down
Sleep 900ms
Down
Sleep 2.5s

# action takes one of four values: pick copy from the list.
Ctrl+O
Sleep 1.8s
Down
Sleep 800ms
Enter
Sleep 1.5s

# Apply the block, then save: validation runs, and the diff asks to confirm.
Ctrl+S
Sleep 1.5s
Ctrl+S
Sleep 3.5s
Enter
Sleep 2s
