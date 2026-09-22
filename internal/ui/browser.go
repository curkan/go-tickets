package ui

import "gotickets/internal/browser"

// openBrowser opens a URL in the default browser
func openBrowser(url string) func() error {
	return func() error {
		return browser.Open(url)
	}
}
