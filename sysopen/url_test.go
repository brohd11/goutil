package sysopen

import (
	"strings"
	"testing"
)

func TestOpenURLRejectsEmpty(t *testing.T) {
	err := OpenURL("")
	if err == nil {
		t.Fatal("empty target should be refused")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error %q should say what was wrong", err)
	}
}
