# Watch Mode

Watch mode monitors source directories in real-time and moves files as they stabilize — that is, after no new writes for `watch.delay`. It is the "set and forget" alternative to running `movelooper` manually.

```bash
movelooper watch
movelooper watch --config /path/to/movelooper.yaml
movelooper watch --category images          # watch only specific categories
movelooper watch --category images,docs
movelooper watch --include-disabled         # include categories with enabled: false
movelooper watch --show-files               # log each file as it is moved
```

---

## How it works

1. movelooper starts a filesystem watcher on every enabled category's `source.path`.
2. When a file event arrives (create or write), the file is added to a pending queue with a timestamp.
3. Every `watch.poll-interval` (default `5s`), pending files are checked. A file graduates from pending to ready when it has not received a new event for at least `watch.delay` (default `5m`).
4. Ready files are processed using the same category rules as the one-shot `movelooper` command: extensions, filters, conflict strategy, organize-by, rename.
5. Every processed batch is recorded in history and can be undone with `movelooper undo`.

---

## Configuration

```yaml
configuration:
  watch:
    delay: 5m           # how long a file must be stable before moving
    poll-interval: 5s   # how often the pending queue is checked
```

| Field | Type | Default | Description |
|---|---|---|---|
| `delay` | duration | `5m` | How long a file must go without a new event before it is considered stable. Accepts Go duration strings: `30s`, `5m`, `1h`. |
| `poll-interval` | duration | `5s` | How often watch re-checks pending files. Keep it shorter than `delay` so stable files are picked up promptly. |

### Tuning delay

- **Large downloads (videos, ISOs):** increase `delay` to `10m` or `15m` to avoid moving files that are still writing.
- **Fast workflows (screenshots, exports):** decrease to `30s` or `1m` if you want near-instant moves.
- **Network shares:** increase `delay` significantly — remote writes can stall without triggering new events.

---

## Limitations

- **`action: archive`** is not processed in watch mode. Categories with `action: archive` are skipped with a warning at startup.
- **Hooks** (`before`/`after`) do not run in watch mode. Use the one-shot `movelooper` command if you need hooks.

---

## Running automatically

### Linux — systemd user service

Create `~/.config/systemd/user/movelooper.service`:

```ini
[Unit]
Description=movelooper watch
After=network.target

[Service]
ExecStart=/usr/local/bin/movelooper watch --config /home/youruser/movelooper.yaml
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

Enable and start:

```bash
systemctl --user enable movelooper
systemctl --user start movelooper
```

Check logs:

```bash
journalctl --user -u movelooper -f
```

### macOS — launchd

Create `~/Library/LaunchAgents/com.movelooper.watch.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.movelooper.watch</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/movelooper</string>
    <string>watch</string>
    <string>--config</string>
    <string>/Users/youruser/movelooper.yaml</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.movelooper.watch.plist
```

### Windows — Task Scheduler

```powershell
$action  = New-ScheduledTaskAction -Execute "movelooper.exe" `
             -Argument "watch --config C:\Users\youruser\movelooper.yaml"
$trigger = New-ScheduledTaskTrigger -AtLogOn
Register-ScheduledTask -TaskName "movelooper" -Action $action -Trigger $trigger -RunLevel Highest
```

> Always use **absolute paths** in scheduled tasks — the working directory is not your home folder when the task runs.
