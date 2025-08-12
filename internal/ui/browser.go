package ui

import (
	"os/exec"
	"runtime"
)

// openBrowser opens a URL in the default browser
func openBrowser(url string) func() error {
	return func() error {
		var cmd string
		var args []string

		switch runtime.GOOS {
		case "windows":
			cmd = "rundll32"
			args = []string{"url.dll,FileProtocolHandler", url}
		case "darwin":
			cmd = "open"
			args = []string{url}
		default: // "linux", "freebsd", "openbsd", "netbsd"
			cmd = "xdg-open"
			args = []string{url}
		}
		return exec.Command(cmd, args...).Start()
	}
}