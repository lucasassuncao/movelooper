# movelooper demo: one messy folder in, an organized tree out.
#
# Records the Downloads folder before, the run itself with --show-files, the
# same folder after, and where everything ended up. The config it uses is
# movelooper.yaml in this directory, which is the same example the README shows.
#
# Run from this directory, with movelooper on your PATH:
#   vhs demo.tape
#
# Everything happens inside ./sandbox, which the tape creates and resets on
# every run. Output paths cannot be absolute: the tape parser rejects a drive
# letter, so demo.gif lands next to this file.

Output demo.gif

Require ttyd
Require ffmpeg

Set Shell         "powershell"
Set FontSize      14
Set Width         1400
Set Height        900
Set Theme         "Catppuccin Mocha"
Set Framerate     30
Set PlaybackSpeed 1.0
Set TypingSpeed   35ms
Set Padding       20

# -- Build the sandbox so the tape can be replayed ----------------------------

Hide
Type "Remove-Item -Recurse -Force sandbox -ErrorAction SilentlyContinue"
Enter
Type "New-Item -ItemType Directory -Force sandbox\Downloads | Out-Null"
Enter
Type "cd sandbox"
Enter
Type "'fake jpeg bytes' | Set-Content Downloads\vacation.jpg"
Enter
Type "'fake png bytes' | Set-Content Downloads\screenshot.png"
Enter
Type "'fake pdf: invoice for march 2026' | Set-Content Downloads\invoice_2026-03.pdf"
Enter
Type "'fake pdf: quarterly report' | Set-Content Downloads\report.pdf"
Enter
Type "'fake pdf: unfinished draft' | Set-Content Downloads\draft_notes.pdf"
Enter
Type "'fake windows installer' | Set-Content Downloads\setup.exe"
Enter
Type "clear"
Enter
Show

# -- Before: six unsorted files -----------------------------------------------

Type "# Downloads before movelooper runs"
Enter
Type "ls Downloads"
Enter
Sleep 4s

# -- Organize -----------------------------------------------------------------

Type "# Run movelooper, listing every file it touches"
Enter
Type "movelooper --config ..\movelooper.yaml --show-files"
Enter
Sleep 7s

Type "clear"
Enter
Sleep 500ms

# -- After: only what no rule claimed is left ---------------------------------

Type "# Downloads after: only what no rule claimed"
Enter
Type "ls Downloads"
Enter
Sleep 4s

# -- Where everything went ----------------------------------------------------

Type "# Where everything went"
Enter
Type "tree /F"
Enter
Sleep 6s
