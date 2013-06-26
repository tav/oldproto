package ui

import "testing"

func TestMessage(t *testing.T) {
	terms := []string{}
	message := "Hello world!"
	parseMsg(message, terms)
	if terms == []strings{"hello", "world"} {
		t.Logf("Passed: %v", terms)
	} else {
		t.Errorf("Failed: %v", terms)
	}
}
