# movelooper demo: one messy folder in, an organized tree out.
#
# Records a single run with --show-files, whose output lists where every file
# went. The config is movelooper.yaml in this directory, the README's example.
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
Set Width         1000
Set Height        380
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
# A home inside the sandbox keeps the run's undo history out of ~/.movelooper.
Type "$env:USERPROFILE = (Get-Location).Path"
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

# -- Organize -----------------------------------------------------------------

Sleep 500ms
Type "movelooper --config ..\movelooper.yaml --show-files"
Sleep 500ms
Enter
Sleep 8s
