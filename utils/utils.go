package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Check whether tmux is running
// non-nil error returned indicate tmux is not running or can not detect
func IsTMUXRunning() error {
	if tmuxEnv := os.Getenv("TMUX"); tmuxEnv == "" {
		return fmt.Errorf("$TMUX not found")
	}

	cmd := exec.Command("pgrep", "tmux")
	out, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return err
	}

	return nil
}

// List tmux sessions, using "tmux list-session"
func ReadTmuxSessions() (*[]string, error) {
	cmd := exec.Command("tmux", "list-session")
	out, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return nil, err
	}

	// format: "0: 1 windows (created Mon Jan  6 21:13:13 2025)"
	outS := string(out)

	lines := strings.Split(outS, "\n")
	sessions := *new([]string)

	re := regexp.MustCompile(`^([^:]*)`)
	for i := range lines {
		if sessionName := re.FindString(lines[i]); strings.TrimSpace(sessionName) != "" {
			sessions = append(sessions, sessionName)
		}
	}

	return &sessions, nil
}

func CurrentTmuxRession() (string, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	out, err := cmd.Output()
	name := strings.Trim(string(out), "\n")
	return name, err
}

// Switch tmux session
func SwitchTmuxSession(name string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", name)
	err := cmd.Run()
	return err
}

func TmuxNewSession(name string) error {
	// -d flag to start a new detached session
	cmd := exec.Command("tmux", "new-session", "-s", name, "-d")
	return cmd.Run()
}

func TmuxHasSession(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	// ignore error "exit status 1"
	out, _ := cmd.CombinedOutput()
	hasSession := strings.TrimRight(string(out), " \n	") == ""
	return hasSession
}

func TmuxRenameSession(oldName, newName string) error {
	cmd := exec.Command("tmux", "rename-session", "-t", oldName, newName)
	return cmd.Run()
}

func TmuxKillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// GetDataFilePath returns the appropriate path for the app data directory based on the OS
func GetAppDataDir(appName string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(appDir, os.ModePerm); err != nil {
		return "", err
	}

	return appDir, nil
}

func GetAppDataFile(appName, filename string) (string, error) {
	appDir, err := GetAppDataDir(appName)
	if err != nil {
		return "", err
	}

	file := filepath.Join(appDir, filename)
	_, err = os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(file)
	}

	return file, err
}

// Read lines of file into []string
func ReadFileLines(path string) (*[]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sessions []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		if text := strings.TrimSpace(scanner.Text()); text != "" {
			sessions = append(sessions, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &sessions, nil
}

// Overwrite file content
func OverwriteFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

// Append file content
func AppendFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func StdErr(msg string) {
	fmt.Fprintf(os.Stderr, "%s\r\n", msg)
}

func StdOut(msg string) {
	fmt.Fprintf(os.Stdout, "%s\r\n", msg)
}

func StdOutf(format string, args ...string) {
	StdOut(fmt.Sprintf(format, args[0:]))
}
