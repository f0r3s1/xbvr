package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/marcsauter/single"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/server"
	"github.com/xbapps/xbvr/ui"
)

var version = "CURRENT"
var commit = "HEAD"
var branch = "master"
var date = "moment ago"

func main() {
	s := single.New("xbvr")
	s.Lock()
	defer s.Unlock()

	systray.Run(onReady, onExit)
}

func onExit() {
	systray.Quit()
	os.Exit(0)
}

func onReady() {
	go func() {
		server.StartServer(version, commit, branch, date)
	}()

	if runtime.GOOS == "windows" {
		iconData, _ := ui.Assets.ReadFile("dist/icons/xbvr-win.ico")
		systray.SetIcon(iconData)
	} else {
		iconData, _ := ui.Assets.ReadFile("dist/icons/xbvr-128.png")
		systray.SetIcon(iconData)
	}
	systray.SetTooltip("XBVR")

	systray.AddSeparator()

	mOpenUI := systray.AddMenuItem("Open UI", "Open UI")
	mOpenConfig := systray.AddMenuItem("Open config folder", "Open config folder")
	mOpenLog := systray.AddMenuItem("Open log file", "Open log file")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit XBVR")

	for {
		select {
		case <-mOpenUI.ClickedCh:
			go openPath(fmt.Sprintf("http://localhost:%v", config.Config.Server.Port))
		case <-mOpenConfig.ClickedCh:
			go openPath(common.AppDir)
		case <-mOpenLog.ClickedCh:
			go openPath(common.AppDir + "/xbvr.log")
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// openPath opens a URL, folder or file with the desktop's default application.
func openPath(target string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
}
