package app

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/luytbq/tmux-harpoon/utils"
	"golang.org/x/term"
)

const (
	cr_pinned = iota
	cr_not_pinned
)

type App struct {
	dataFile          string
	allSessions       []string
	pinnedSessions    []string
	notPinnedSessions []string
	lenAll            int
	lenPinned         int
	lenNotPinned      int
	index             int
	cursorRegion      int
	Debug             bool
	debugInfo         string
	termOldState      *term.State
}

func NewApp() *App {
	userDataFile, err := utils.GetDataFilePath()
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}

	app := &App{
		dataFile:     userDataFile,
		cursorRegion: cr_pinned,
	}

	pinnedSessions, err := utils.ReadFileLines(app.dataFile)
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}
	app.pinnedSessions = *pinnedSessions

	allSessions, err := utils.ReadTmuxSessions()
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}
	app.allSessions = *allSessions

	app.process()

	currentName, err := utils.CurrentTmuxRession()
	if err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return nil
	}

	for i, v := range app.pinnedSessions {
		if v == currentName {
			app.index = i
			break
		}
	}

	for i, v := range app.notPinnedSessions {
		if v == currentName {
			app.index = i + app.lenPinned
			break
		}
	}

	return app
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
		idx := slices.IndexFunc(a.pinnedSessions, func(s string) bool { return s == v })
		if idx < 0 {
			a.notPinnedSessions = append(a.notPinnedSessions, v)
		}
	}

	a.lenAll = len(a.allSessions)
	a.lenPinned = len(a.pinnedSessions)
	a.lenNotPinned = len(a.notPinnedSessions)

	if a.lenPinned == 0 || a.index >= a.lenPinned {
		a.cursorRegion = cr_not_pinned
	} else {
		a.cursorRegion = cr_pinned
	}
}

func (a *App) update() {
	a.process()
	a.print()
	a.savePinnedSessions()
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

	if a.Debug && a.debugInfo != "" {
		fmt.Printf("\r\n%s", a.debugInfo)
		fmt.Printf("\r\n%+v", a)
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
}

func (a *App) pinSession() {
	a.pinnedSessions = append(a.pinnedSessions, a.notPinnedSessions[a.index-a.lenPinned])
	a.update()
}

func (a *App) unpinSession() {
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
		// shift line between the two up
		copy(a.pinnedSessions[target+1:a.index+1], a.pinnedSessions[target:a.index])
	} else {
		// shift line between the two down
		copy(a.pinnedSessions[a.index:target], a.pinnedSessions[a.index+1:target+1])
	}
	a.pinnedSessions[target] = tmp
	a.index = target
}

func (a *App) savePinnedSessions() {
	if err := utils.OverwriteFile(a.dataFile, strings.Join(a.pinnedSessions, "\n")); err != nil {
		utils.StdErr(err.Error())
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

func (a *App) Interact() {
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
		buf := make([]byte, 3)
		_, err := os.Stdin.Read(buf)
		if err != nil {
			utils.StdErr(err.Error())
			os.Exit(1)
			return
		}

		key := buf[0]
		key1 := buf[1]
		key2 := buf[2]

		switch {
		case key == 3: // Ctrl-C
			a.shutdown()
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
		case key == '\r': // enter
			flushStdin()
			a.switchSession()
			if !a.Debug {
				return
			}

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
		}

		a.debugInfo = fmt.Sprintf("\r\nkey='%d' '%d' '%d'", key, key1, key2)
		a.update()
	}
}

// Clear the screen
func flushStdin() {
	fmt.Print("\033[H\033[2J")
}
