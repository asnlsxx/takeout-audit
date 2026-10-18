package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDedicatedCommandHasNoMutationOrUpdater(t *testing.T) {
	root := command()
	if len(root.Commands) != 0 {
		t.Fatal("dedicated binary must not expose admin subcommands")
	}
	var output bytes.Buffer
	root.Writer = &output
	if err := root.Run(context.Background(), []string{"takeout-audit", "--help"}); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--inventory-only", "--out", "--config"} {
		if !strings.Contains(output.String(), flag) {
			t.Errorf("missing %s", flag)
		}
	}
}
