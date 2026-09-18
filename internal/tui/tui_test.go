package tui

import (
	"reflect"
	"testing"

	"github.com/Viswesh-G/scope/internal/search"
)

func TestSearchArgsExposeTUIControls(t *testing.T) {
	args, err := searchArgs(model{
		queryInput:   "TODO",
		pathInput:    ".",
		contextInput: "2",
		flagsInput:   "-g *.go",
		ignoreCase:   true,
		resultMode:   modeHotspots,
	})
	if err != nil {
		t.Fatalf("searchArgs returned error: %v", err)
	}

	want := []string{"search", "-p", "TODO", "--path", ".", "--json", "-q",
		"--ignore-case", "--context", "2", "-g", "*.go"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestSearchArgsRejectInvalidContext(t *testing.T) {
	_, err := searchArgs(model{queryInput: "x", pathInput: ".", contextInput: "-1"})
	if err == nil {
		t.Fatal("expected invalid context error")
	}
}

func TestBuildHotspotsRanksFiles(t *testing.T) {
	matches := []search.JSONMatch{
		{File: "b.go"}, {File: "a.go"}, {File: "b.go"},
	}
	got := buildHotspots(matches)
	want := []hotspot{{file: "b.go", matches: 2}, {file: "a.go", matches: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hotspots = %#v, want %#v", got, want)
	}
}
