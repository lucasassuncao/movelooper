# Movelooper

## Requirements

You need a configuration file named `movelooper.yaml` in the same directory as movelooper.

### Example: Config file (movelooper.yaml) 

```yaml

configuration:
  output: console # Default is console (console, log or file)
  log-file: "<path_to_logfile>"
  log-level: debug # trace, debug, info, warn/warning, error or fatal
  show-caller: false

categories:
  images:
    extensions: ["jpg", "jpeg", "png", "gif", "webp"]
    source: "C:\\Users\\lucas\\Downloads\\"
    destination: "C:\\Users\\lucas\\Downloads\\Media\\Images\\"

  audio:
    extensions: ["mp3"]
    source: "C:\\Users\\lucas\\Downloads\\"
    destination: "C:\\Users\\lucas\\Downloads\\Media\\Audio\\"

   # Add more categories as you want

```

### Example: Help

```bash

root ➜ /workspaces/movelooper (main) $ go run .
Long description of newMoveLooper

Usage:
  movelooper [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  move        Moves files to their respective destination directories based on configured categories
  preview     Displays a preview of files to be moved based on configured categories

Flags:
  -h, --help               help for movelooper
  -l, --log-level string   Specify the log level
  -o, --output string      Specify the output (console, log or file)
      --show-caller        Show caller information

Use "movelooper [command] --help" for more information about a command.

```

### Example: Usage

#### Previewing files
```powershell

💀 lucas@Milkyway 📂 C:\Users\lucas\go\src\movelooper>.\movelooper.exe preview -o console -l debug
2025-02-24 19:17:17 INFO  Starting newMoveLooper
2025-02-24 19:17:17 INFO  No .jpg file(s) to move
2025-02-24 19:17:17 INFO  No .jpeg file(s) to move
2025-02-24 19:17:17 INFO  No .png file(s) to move
2025-02-24 19:17:17 INFO  No .gif file(s) to move
2025-02-24 19:17:17 INFO  No .webp file(s) to move
2025-02-24 19:17:17 WARN  1 file .mp3 to move
2025-02-24 19:17:17 INFO  No .mp4 file(s) to move
2025-02-24 19:17:17 INFO  No .pdf file(s) to move
2025-02-24 19:17:17 INFO  No .txt file(s) to move
2025-02-24 19:17:17 INFO  No .docx file(s) to move
2025-02-24 19:17:17 INFO  No .pptx file(s) to move
2025-02-24 19:17:17 INFO  No .zip file(s) to move
2025-02-24 19:17:17 INFO  No .rar file(s) to move
2025-02-24 19:17:17 INFO  No .7z file(s) to move
2025-02-24 19:17:17 INFO  No .exe file(s) to move
2025-02-24 19:17:17 INFO  No .msi file(s) to move
2025-02-24 19:17:17 INFO  No .apk file(s) to move
2025-02-24 19:17:17 INFO  No .pkg file(s) to move
2025-02-24 19:17:17 INFO  No .iso file(s) to move
2025-02-24 19:17:17 INFO  No .ttf file(s) to move
2025-02-24 19:17:17 INFO  No .otf file(s) to move

```

#### Moving files
```powershell

💀 lucas@Milkyway 📂 C:\Users\lucas\go\src\movelooper>.\movelooper.exe move -o console -l debug
2025-02-24 19:18:25 INFO  Starting newMoveLooper
2025-02-24 19:18:25 INFO  No .pdf file(s) to move
2025-02-24 19:18:25 INFO  No .txt file(s) to move
2025-02-24 19:18:25 INFO  No .docx file(s) to move
2025-02-24 19:18:25 INFO  No .pptx file(s) to move
2025-02-24 19:18:25 INFO  No .zip file(s) to move
2025-02-24 19:18:25 INFO  No .rar file(s) to move
2025-02-24 19:18:25 INFO  No .7z file(s) to move
2025-02-24 19:18:25 INFO  No .exe file(s) to move
2025-02-24 19:18:25 INFO  No .msi file(s) to move
2025-02-24 19:18:25 INFO  No .apk file(s) to move
2025-02-24 19:18:25 INFO  No .pkg file(s) to move
2025-02-24 19:18:25 INFO  No .iso file(s) to move
2025-02-24 19:18:25 INFO  No .ttf file(s) to move
2025-02-24 19:18:25 INFO  No .otf file(s) to move
2025-02-24 19:18:25 INFO  No .jpg file(s) to move
2025-02-24 19:18:25 INFO  No .jpeg file(s) to move
2025-02-24 19:18:25 INFO  No .png file(s) to move
2025-02-24 19:18:25 INFO  No .gif file(s) to move
2025-02-24 19:18:25 INFO  No .webp file(s) to move
2025-02-24 19:18:25 WARN  1 file .mp3 to move
2025-02-24 19:18:25 INFO  successfully moved file
                      ├ source: C:\Users\lucas\Downloads\Please take everything I have.mp3
                      └ destination: C:\Users\lucas\Downloads\Media\Audio\mp3\Please take everything I have.mp3
2025-02-24 19:18:25 INFO  No .mp4 file(s) to move

```

## Execution Flow

1. Initialize Command (RootCmd):
   1. Sets up a movelooper command with descriptions and a PersistentPreRun function.
2. In PersistentPreRun:
   1. Loads settings from a movelooper.yaml file.
   2. If no logger exists, it sets up a logger.
   3. Ensures flags (output, show-caller, log-level) are correctly set, either from the command line or the config file.
3. Defines persistent flags for the command:
   1. show-caller, log-level, and output.
4. Links flags to Viper keys for configuration support.
5. Adds two subcommands to the root command:
   1. PreviewCmd(m): Preview functionality.
   2. MoveCmd(m): Move functionality.
6. The configured movelooper command is returned and ready to execute.