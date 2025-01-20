# TMUX Session Manager

TMUX Session Manager is a command-line tool for managing your TMUX sessions. This app allows you to pin, reorder, rename, and switch between TMUX sessions with an interactive interface or simple commands.

## Features

- **Interactive Mode:** Navigate, pin, and reorder sessions using keyboard shortcuts.
- **Pinned Sessions Management:** Mark frequently used sessions as pinned for quick access.
- **Session Switching:** Easily switch to any pinned session.
- **Session Renaming:** Rename TMUX sessions interactively.
- **Create New Sessions:** Quickly create and switch to a new TMUX session.

## Requirements

- **TMUX** installed and running.
- **Go** installed to build the application.

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/luytbq/tmux-session-manager.git
   cd tmux-session-manager
   ```

2. Build the binary:

   ```bash
   go build -o tmux-session-manager main.go
   ```

3. Move the binary to a directory in your PATH:

   ```bash
   mv tmux-session-manager /usr/local/bin/
   chmod +x /usr/local/bin/tmux-session-manager
   ```

4. Add tmux shortcuts

```bash
# add this to your .tmux.conf file
  # replace prefix-s with tmux-session-list
  bind-key -r s run-shell "tmux neww /usr/bin/tmux-session-list"

  # switch to pinned sessions
  bind-key -r j run-shell "tmux neww /usr/bin/tmux-session-list switch 1"
  bind-key -r k run-shell "tmux neww /usr/bin/tmux-session-list switch 2"
  bind-key -r u run-shell "tmux neww /usr/bin/tmux-session-list switch 3"
  bind-key -r i run-shell "tmux neww /usr/bin/tmux-session-list switch 4"
```

## Usage

### Basic Commands

- **Interactive Mode:**

  ```bash
  tmux-session-manager
  ```

  or

  ```bash
  tmux-session-manager it
  ```

- **List Pinned Sessions:**

  ```bash
  tmux-session-manager list
  ```

- **Switch to a Pinned Session:**

  ```bash
  tmux-session-manager switch <pinned index>
  ```

  Replace `<pinned index>` with the index of the pinned session (e.g., `1`, `2`, `3`, etc.).

### Interactive Mode Shortcuts

- `j` / `k`: Move focus down/up.
- `J` / `K`: Swap session with the one below/above.
- `p`: Pin the currently selected session.
- `P`: Unpin the currently selected session.
- `n`: Create a new session interactively.
- `r`: Rename the currently selected session interactively.
- `1-9`: Switch to a pinned session by index.
- `Shift-1` to `Shift-9`: Reorder pinned sessions to a specific position.
- `Enter`: Switch to the currently selected session.
- `Esc`: Exit interactive mode.
- `Ctrl-C`: Exit the application.

## Error Handling

- If TMUX is not running, the application will output an error and exit.
- Invalid inputs for commands such as `switch` will result in a descriptive error message.

---

Enjoy managing your TMUX sessions with ease!
