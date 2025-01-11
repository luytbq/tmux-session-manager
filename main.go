package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/luytbq/tmux-session-list/app"
	"github.com/luytbq/tmux-session-list/utils"
)

func main() {
	if err := utils.IsTMUXRunning(); err != nil {
		utils.StdErr(err.Error())
		os.Exit(1)
		return
	}

	args := os.Args

	var cmd string
	if len(args) > 1 {
		cmd = args[1]
	}

	app := app.NewApp()

	switch cmd {
	case "", "it", "interactive":
		app.Interactive()
	case "list":
		app.PrintPinned()
	case "switch":
		var argTarget string
		if len(args) > 2 {
			argTarget = args[2]
		}
		target, err := strconv.ParseInt(argTarget, 10, 0)
		if err != nil {
			panic("invalid target " + argTarget)
		}
		app.SwitchToPinned(int(target))
	default:
		help()
		os.Exit(1)
	}

}

func help() {
	fmt.Fprint(os.Stdout, "Usage: tmux-pin [it | interactive | switch] [<pinned index>]\n")
}
