package app

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/luytbq/tmux-session-list/config"
	"github.com/luytbq/tmux-session-list/log"
	"github.com/luytbq/tmux-session-list/utils"
	"golang.org/x/term"
)

const (
	cr_pinned = iota
	cr_not_pinned
)

var pinnedSessionsFile string

func initFiles() error {
	file, err := utils.GetAppDataFile(config.AppName, config.PinnedSessionsFile)
	if err != nil {
		return err
	}
	pinnedSessionsFile = file

	return nil
}

type App struct {
	allSessions       []string
	pinnedSessions    []string
	notPinnedSessions []string
	lenAll            int
	lenPinned         int
	lenNotPinned      int
	index             int
	cursorRegion      int
	termOldState      *term.State
}

func NewApp() *App {
	err := initFiles()
	if err != nil {
		utils.StdErr(err.Error())
	}

	app := &App{
		cursorRegion: cr_pinned,
	}

	pinnedSessions, err := utils.ReadFileLines(pinnedSessionsFile)
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}
	app.pinnedSessions = *pinnedSessions

	app.getAllSessions()

	app.calculateCursorRegion()

	app.process()

	currentName, err := utils.CurrentTmuxRession()
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}

	for i, v := range app.pinnedSessions {
		if v == currentName {
			app.move(i)
			break
		}
	}

	for i, v := range app.notPinnedSessions {
		if v == currentName {
			app.index = i + app.lenPinned
			break
		}
	}

	log.Log(log.Info, config.AppName+" initialized")
	return app
}

func (a *App) getAllSessions() {
	allSessions, err := utils.ReadTmuxSessions()
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return
	}
	a.allSessions = *allSessions
}

func (a *App) process() {

	tmp := *new([]string)
	for _, v := range a.pinnedSessions {
		idx := slices.Index(a.allSessions, v)
		if idx > -1 {
			tmp = append(tmp, v)
		}
	}
	a.pinnedSessions = tmp

	a.notPinnedSessions = *new([]string)
	for _, v := range a.allSessions {
		idx := slices.Index(a.pinnedSessions, v)
		if idx < 0 {
			a.notPinnedSessions = append(a.notPinnedSessions, v)
		}
	}

	a.lenAll = len(a.allSessions)
	a.lenPinned = len(a.pinnedSessions)
	a.lenNotPinned = len(a.notPinnedSessions)
}

func (a *App) update() {
	a.process()
	a.print()
	a.savePinnedSessions()
	a.calculateCursorRegion()
}

func (a *App) PrintPinned() {
	for i, line := range a.pinnedSessions {
		cursor := " "
		if i == a.index {
			cursor = ">"
		}

		fmt.Printf("%s [%d] %s\r\n", cursor, i, line)
	}
}

func (a *App) print() {
	flushStdin()
	fmt.Printf("*** Pinned Sessions (%d/%d) ***\r\n", a.lenPinned, a.lenAll)
	a.PrintPinned()

	fmt.Printf("\r\n*** Other Sessions (%d/%d) ***\r\n", a.lenNotPinned, a.lenAll)
	for i, line := range a.notPinnedSessions {
		cursor := " "
		if i+a.lenPinned == a.index {
			cursor = ">"
		}

		fmt.Printf("%s [%d] %s\r\n", cursor, i, line)
	}
}

func (a *App) move(index int) {
	if index < 0 {
		a.index = a.lenAll - 1
	} else if index >= a.lenAll {
		a.index = 0
	} else {
		a.index = index
	}

	a.calculateCursorRegion()
}

func (a *App) calculateCursorRegion() {
	log.Log(log.Trace, "calculateCursorRegion before", a)
	defer log.Log(log.Trace, "calculateCursorRegion after ", a)
	if a.lenPinned == 0 || a.index >= a.lenPinned {
		a.cursorRegion = cr_not_pinned
	} else {
		a.cursorRegion = cr_pinned
	}
}

func (a *App) pinSession() {
	log.Log(log.Trace, "pinSession before", a)
	defer log.Log(log.Trace, "pinSession after ", a)
	if a.cursorRegion != cr_not_pinned {
		return
	}
	a.pinnedSessions = append(a.pinnedSessions, a.getSelectedSession())
	a.update()
}

func (a *App) unpinSession() {
	if a.cursorRegion != cr_pinned {
		return
	}
	a.pinnedSessions = append(a.pinnedSessions[:a.index], a.pinnedSessions[a.index+1:]...)
	a.update()
}

func (a *App) getSelectedSession() string {
	if a.cursorRegion == cr_pinned {
		return a.pinnedSessions[a.index]
	} else {
		return a.notPinnedSessions[a.index-a.lenPinned]
	}
}

// Switch to session at index
func (a *App) switchSession() {
	err := utils.SwitchTmuxSession(a.getSelectedSession())
	if err != nil {
		utils.StdErr(err.Error())
	}
}

func (a *App) swap(target int) {
	if target < 0 || target >= len(a.pinnedSessions) {
		return
	}
	a.pinnedSessions[a.index], a.pinnedSessions[target] = a.pinnedSessions[target], a.pinnedSessions[a.index]
	a.index = target
}

// move focused line to target
func (a *App) reOrder(target int) {
	if target < 0 || target >= len(a.pinnedSessions) || target == a.index {
		return
	}
	tmp := a.pinnedSessions[a.index]
	if a.index > target {
		// shift lines between the two up
		copy(a.pinnedSessions[target+1:a.index+1], a.pinnedSessions[target:a.index])
	} else {
		// shift lines between the two down
		copy(a.pinnedSessions[a.index:target], a.pinnedSessions[a.index+1:target+1])
	}
	a.pinnedSessions[target] = tmp
	a.index = target
}

func (a *App) savePinnedSessions() {
	pinnedSessionsFile, err := utils.GetAppDataFile(config.AppName, config.PinnedSessionsFile)
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return
	}
	err = utils.OverwriteFile(pinnedSessionsFile, strings.Join(a.pinnedSessions, "\n"))
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
	}
}

func (a *App) shutdown() {
	_ = term.Restore(int(os.Stdin.Fd()), a.termOldState)
	os.Exit(1)
}

func (a *App) SwitchToPinned(target int) {
	if target >= a.lenPinned {
		utils.StdErr(fmt.Sprintf("Session #%d not found", target))
		os.Exit(1)
		return
	}

	a.index = target
	a.switchSession()
}

func (a *App) Interactive() {
	// Set up terminal for raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		utils.StdErr(err.Error())
		a.shutdown()

	}
	a.termOldState = oldState
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	// Handle graceful exit on Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		a.shutdown()
	}()

	a.print()
	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err != nil {
			utils.StdErr(err.Error())
			os.Exit(1)
			return
		}

		key := buf[0]

		log.Log(log.Trace, fmt.Sprintf("key received: int(key) = '%d', string(key)='%s'", int(key), string(key)), a)

		switch {
		case key == 3: // Ctrl-C
			a.shutdown()
			return
		case key >= 48 && key <= 57: // 0->9
			a.SwitchToPinned(int(key) - 48)
			return
		case key == 'j': // "j" focus down
			a.move(a.index + 1)
		case key == 'k': // "k" focus up
			a.move(a.index - 1)
		case key == 'J': // "J" swap down
			a.swap(a.index + 1)
		case key == 'K': // "K" swap up
			a.swap(a.index - 1)
		case key == 27: // "esc"
			flushStdin()
			return
		case key == 'n':
			a.NewSessionInteractive()
		case key == 'r':
			a.RenameSessionInteractive()
		case key == '\r': // enter
			flushStdin()
			a.switchSession()
			return

		// cursor at "other sessions"
		case a.cursorRegion == cr_not_pinned:
			switch key {
			case 'p':
				a.pinSession()
			}

		// cursor at "pinned sessions"
		case a.cursorRegion == cr_pinned:
			{
				switch key {
				case 'P':
					a.unpinSession()

				// "Shift-0" -> "Shift-9" put focused line to index
				case ')':
					a.reOrder(0)
				case '!':
					a.reOrder(1)
				case '@':
					a.reOrder(2)
				case '#':
					a.reOrder(3)
				case '$':
					a.reOrder(4)
				case '%':
					a.reOrder(5)
				case '^':
					a.reOrder(6)
				case '&':
					a.reOrder(7)
				case '*':
					a.reOrder(8)
				case '(':
					a.reOrder(9)
				// end

				// case '\r': // "Enter" save file then return selected
				// 	if a.index >= 0 && a.index < len(a.pinnedSessions) {
				// 		a.SavePinnedSession()
				// 		flushStdin()
				// 		fmt.Printf("%s\r\n", a.pinnedSessions[a.index])
				// 	}
				// 	return
				default:
				}
			}

		default:
		}

		a.update()
	}
}

func (a *App) PromptInput(prompt string) (string, error) {
	fmt.Print(prompt)

	input := ""
	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err != nil {
			utils.StdErr(err.Error())
			return "", err
		}
		key := buf[0]

		switch key {
		case '\r':
			return strings.TrimSpace(input), nil
		case 3, 27: // Ctrl-C, esc
			return "", fmt.Errorf("canceled")
		default:
			input += string(key)
			fmt.Print(string(key))
		}
	}
}

func (a *App) RenameSessionInteractive() {
	oldName := a.getSelectedSession()
	newName, err := a.PromptInput(fmt.Sprintf("Rename '%s' to: ", oldName))
	log.Log(log.Trace, "RenameSessionInteractive", newName, err)
	if err != nil {
		fmt.Print(err.Error() + "\r\n")
		return
	}
	log.Log(log.Trace, "RenameSessionInteractive", oldName)
	if newName == "" || newName == oldName {
		return
	}
	a.RenameSession(oldName, newName)
	if a.cursorRegion == cr_pinned {
		idx := slices.Index(a.pinnedSessions, oldName)
		if idx > 0 {
			a.pinnedSessions[idx] = newName
		}
	}
	a.getAllSessions()
	a.process()
}

func (a *App) RenameSession(oldName, newName string) {
	hasSession := utils.TmuxHasSession(newName)
	log.Log(log.Info, fmt.Sprintf("RenameSession '%s' -> '%s' hasSession=%t", oldName, newName, hasSession))
	if hasSession {
		utils.StdOut(fmt.Sprintf("Session with name '%s' already existed", newName))
		return
	}
	err := utils.TmuxRenameSession(oldName, newName)
	if err != nil {
		utils.StdOut(err.Error())
		return
	}
}

func (a *App) NewSessionInteractive() {
	// a.print()
	name, err := a.PromptInput("Enter new session name: ")
	if err != nil {
		fmt.Print(err.Error() + "\r\n")
		return
	}

	if name == "" {
		return
	}
	a.NewSession(name)
	a.getAllSessions()
	a.process()
}

func (a *App) NewSession(name string) {
	err := utils.TmuxNewSession(name)
	if err != nil {
		utils.StdOut(err.Error())
		return
	}
}

// Clear the screen
func flushStdin() {
	fmt.Print("\033[H\033[2J")
}
