# Movelooper

`movelooper` is a CLI tool for organizing and moving files from source directories to destination directories, based on configurable categories. The tool supports two main commands:

- **Preview:** A dry-run to show what files would be moved without actually performing the operation.
- **Move:** Actually moves files from source directories to destination directories, placing them in subdirectories based on their extensions.

## Requirements

You need a configuration file named `movelooper.yaml` in the same directory as movelooper.

### Example: Config file (movelooper.yaml) 

```yaml

configuration:
  output: console
  log-file: "C:\\Users\\lucas\\Desktop\\newMoveLooper.log"
  log-level: debug
  show-caller: false

categories:
  - name: "images"
    extensions: ["jpg", "jpeg", "png", "gif", "webp"]
    source: "C:\\Users\\lucas\\Downloads\\"
    destination: "C:\\Users\\lucas\\Downloads\\Media\\Images\\"

  - name: "audio"
    extensions: ["mp3"]
    source: "C:\\Users\\lucas\\Downloads\\"
    destination: "C:\\Users\\lucas\\Downloads\\Media\\Audio\\"

   # Add more categories as you want

```

### Example: Help

```bash

root ➜ /workspaces/movelooper (main) $ go run .
movelooper is a CLI tool for organizing and moving files from source directories to destination directories, based on configurable categories

Usage:
  movelooper [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  move        Moves files to their respective destination directories based on configured categories
  preview     Displays a preview of files to be moved based on configured categories (dry-run)

Flags:
  -h, --help               help for movelooper
  -l, --log-level string   Specify the log level (trace, debug, info, warn/warning, error, fatal)
  -o, --output string      Specify the output (console, log/file or both)
      --show-caller        Show caller information

Use "movelooper [command] --help" for more information about a command.

```

### Example: Usage

#### Previewing files
```powershell

💀 lucas@Milkyway 📂 C:\Users\lucas\go\src\movelooper>.\movelooper.exe preview -o console -l debug
2025-02-24 22:27:12 INFO  Starting preview mode 
2025-02-24 22:27:12 INFO  No jpg file(s) to move 
2025-02-24 22:27:12 INFO  No jpeg file(s) to move 
2025-02-24 22:27:12 INFO  No png file(s) to move 
2025-02-24 22:27:12 INFO  No gif file(s) to move 
2025-02-24 22:27:12 INFO  No webp file(s) to move 
2025-02-24 22:27:12 INFO  No mp3 file(s) to move 
2025-02-24 22:27:12 INFO  No mp4 file(s) to move 
2025-02-24 22:27:12 INFO  No pdf file(s) to move 
2025-02-24 22:27:12 INFO  No txt file(s) to move
2025-02-24 22:27:12 INFO  No docx file(s) to move
2025-02-24 22:27:12 INFO  No pptx file(s) to move
2025-02-24 22:27:12 INFO  No zip file(s) to move
2025-02-24 22:27:12 INFO  No rar file(s) to move
2025-02-24 22:27:12 INFO  No 7z file(s) to move
2025-02-24 22:27:12 INFO  No exe file(s) to move
2025-02-24 22:27:12 WARN  1 file msi to move
2025-02-24 22:27:12 INFO  No apk file(s) to move
2025-02-24 22:27:12 INFO  No pkg file(s) to move
2025-02-24 22:27:12 INFO  No iso file(s) to move
2025-02-24 22:27:12 INFO  No ttf file(s) to move
2025-02-24 22:27:12 INFO  No otf file(s) to move

```

#### Moving files
```powershell

💀 lucas@Milkyway 📂 C:\Users\lucas\go\src\movelooper>.\movelooper.exe move -o console -l debug
2025-02-24 22:27:19 INFO  Starting move mode
2025-02-24 22:27:19 INFO  No jpg file(s) to move
2025-02-24 22:27:19 INFO  No jpeg file(s) to move
2025-02-24 22:27:19 INFO  No png file(s) to move
2025-02-24 22:27:19 INFO  No gif file(s) to move
2025-02-24 22:27:19 INFO  No webp file(s) to move
2025-02-24 22:27:19 INFO  No mp3 file(s) to move
2025-02-24 22:27:19 INFO  No mp4 file(s) to move
2025-02-24 22:27:19 INFO  No pdf file(s) to move
2025-02-24 22:27:19 INFO  No txt file(s) to move
2025-02-24 22:27:19 INFO  No docx file(s) to move
2025-02-24 22:27:19 INFO  No pptx file(s) to move
2025-02-24 22:27:19 INFO  No zip file(s) to move
2025-02-24 22:27:19 INFO  No rar file(s) to move
2025-02-24 22:27:19 INFO  No 7z file(s) to move
2025-02-24 22:27:19 INFO  No exe file(s) to move
2025-02-24 22:27:19 WARN  1 file msi to move
2025-02-24 22:27:19 INFO  successfully moved file
                      ├ source: C:\Users\lucas\Downloads\PowerShell-7.5.0-win-x64.msi
                      └ destination: C:\Users\lucas\Downloads\Installers\msi\PowerShell-7.5.0-win-x64.msi
2025-02-24 22:27:19 INFO  No apk file(s) to move
2025-02-24 22:27:19 INFO  No pkg file(s) to move
2025-02-24 22:27:19 INFO  No iso file(s) to move
2025-02-24 22:27:19 INFO  No ttf file(s) to move
2025-02-24 22:27:19 INFO  No otf file(s) to move

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

### Subcommand Workflow

1. **Preview Command:**
  - **Objective:** Displays a preview of the files that would be moved without making any changes.
  - **Process:**
    - The tool scans the source directories for each configured category.
    - For each category, it checks for files matching the specified extensions.
    - It counts the matching files and logs the number of files to be moved for each extension.
    - This command doesn't perform the file movement; it’s a verification step.
2. **Move Command:**
  - **Objective:** Actually moves the files to the corresponding destination directories.
  - **Process:**
    - Similar to the preview command, it scans the source directories for files matching the specified extensions.
    - For each extension, it moves the files into subdirectories named after their extensions within the destination directory.
    - The tool creates the necessary destination directories if they don’t exist.

<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# movelooper

```go
import "movelooper"
```

## Index



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
