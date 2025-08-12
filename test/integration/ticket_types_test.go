package integration

import (
	"testing"

	"gotickets/pkg/gotickets"
)

func TestTicket_FilterValue_And_GetDescription(t *testing.T) {
	ticket := gotickets.Ticket{ID: 1, Title: "Test", URL: "https://example.com/x"}
	if got, want := ticket.FilterValue(), "Test https://example.com/x"; got != want {
		t.Errorf("FilterValue() = %q, want %q", got, want)
	}
	if got, want := ticket.GetDescription(), ticket.URL; got != want {
		t.Errorf("GetDescription() = %q, want %q", got, want)
	}
}