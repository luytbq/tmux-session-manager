package app

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/luytbq/tmux-session-manager/config"
	"github.com/luytbq/tmux-session-manager/log"
	"github.com/luytbq/tmux-session-manager/utils"
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

	log.Info(config.AppName + " initialized")
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

		fmt.Printf("%s [%d] %s\r\n", cursor, i+1, line)
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

		fmt.Printf("%s [%d] %s\r\n", cursor, i+1, line)
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
	if a.lenPinned == 0 || a.index >= a.lenPinned {
		a.cursorRegion = cr_not_pinned
	} else {
		a.cursorRegion = cr_pinned
	}
}

func (a *App) pinSession() {
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
	log.Debug(fmt.Sprintf("switchSession begin | a.index=%d", a.index))
	defer log.Debug("switchSession end")
	err := utils.SwitchTmuxSession(a.getSelectedSession())
	if err != nil {
		log.Error(err.Error())
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
	log.Debug(fmt.Sprintf("SwitchToPinned begin | target=%d", target))
	defer log.Debug("SwitchToPinned end")
	if target >= a.lenPinned || target < 0 {
		log.Error(fmt.Sprintf("Session #%d not found", target))
		os.Exit(1)
		return
	}

	a.index = target
	a.calculateCursorRegion()
	a.switchSession()
}

func (a *App) SwitchToName(name string) error {
	log.Debug("Enter new session")
	return utils.SwitchTmuxSession(name)
}

// Enter terminal raw mode in order to perform interactive actions
func (a *App) enterTermRawMode() {
	// Set up terminal for raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		utils.StdErr(err.Error())
		a.shutdown()
	}
	a.termOldState = oldState
}

// Exit terminal raw mode
func (a *App) exitTermRawMode() {
	_ = term.Restore(int(os.Stdin.Fd()), a.termOldState)
}

func (a *App) Interactive() {

	a.enterTermRawMode()
	defer a.exitTermRawMode()

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

		log.Trace(fmt.Sprintf("key received: int(key) = '%d', string(key)='%s'", int(key), string(key)), a)

		switch {
		case key == 3: // Ctrl-C
			a.shutdown()
			return
		case key >= 49 && key <= 57: // 1->9
			a.SwitchToPinned(int(key) - 49)
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
		case key == 'd':
			a.KillSessionInteractive()
		case key == '\r': // enter
			flushStdin()
			a.switchSession()
			return

		// cursor at "other sessions"
		case a.cursorRegion == cr_not_pinned:
			switch key {
			case 'p':
				a.pinSession()
			default:
				log.Trace(fmt.Sprintf("cursor = '%d' unknown key received: int(key) = '%d', string(key)='%s'", a.cursorRegion, int(key), string(key)))
			}

		// cursor at "pinned sessions"
		case a.cursorRegion == cr_pinned:
			{
				switch key {
				case 'P':
					a.unpinSession()

				// "Shift-1" -> "Shift-9" put focused line to index
				case '!':
					a.reOrder(0)
				case '@':
					a.reOrder(1)
				case '#':
					a.reOrder(2)
				case '$':
					a.reOrder(3)
				case '%':
					a.reOrder(4)
				case '^':
					a.reOrder(5)
				case '&':
					a.reOrder(6)
				case '*':
					a.reOrder(7)
				case '(':
					a.reOrder(8)
				// end

				default:
					log.Trace(fmt.Sprintf("cursor = '%d' unknown key received: int(key) = '%d', string(key)='%s'", a.cursorRegion, int(key), string(key)))
				}
			}

		default:
		}

		a.update()
	}
}

func (a *App) PromptInput(prompt string) (string, error) {
	a.exitTermRawMode()
	defer a.enterTermRawMode()

	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin) // Create a reader to read user input
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err.Error())
	}

	return strings.TrimSpace(input), err
}

func (a *App) RenameSessionInteractive() {
	oldName := a.getSelectedSession()
	newName, err := a.PromptInput(fmt.Sprintf("Rename '%s' to: ", oldName))
	log.Trace("RenameSessionInteractive", newName, err)
	if err != nil {
		fmt.Print(err.Error() + "\r\n")
		return
	}
	log.Trace("RenameSessionInteractive", oldName)
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
	log.Info(fmt.Sprintf("RenameSession '%s' -> '%s' hasSession=%t", oldName, newName, hasSession))
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

func (a *App) KillSessionInteractive() {
	fmt.Printf("Press Enter to kill session '%s'", a.getSelectedSession())
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return
	}
	if buf[0] == '\r' {
		err = a.killSession()
		if err != nil {
			utils.StdErr(err.Error())
			os.Exit(1)
			return
		}
	}
	a.getAllSessions()
	a.process()
}

func (a *App) killSession() error {
	return utils.TmuxKillSession(a.getSelectedSession())
}

func (a *App) NewSessionInteractive() {
	// a.print()
	log.Debug("NewSessionInteractive")
	name, err := a.PromptInput("Enter new session name: ")
	if err != nil {
		fmt.Print(err.Error() + "\r\n")
		return
	}

	if name == "" {
		return
	}
	a.NewSession(name)

	fmt.Printf("Press Enter to switch to '%s'", name)
	buf := make([]byte, 1)
	_, err = os.Stdin.Read(buf)
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return
	}
	if buf[0] == '\r' {
		err = a.SwitchToName(name)
		if err != nil {
			utils.StdErr(err.Error())
			os.Exit(1)
			return
		}
		a.shutdown()
	}

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
