package state

import (
	"os"
	"testing"
)

func TestWd(t *testing.T) {

	path := "/some/test/path"

	os.Setenv("SNAP_DATA", path)

	if Wd() != path {
		t.Errorf("want %s, got %s", path, Wd())
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("want nil, got %s", err)
	}

	if cwd == Wd() {
		t.Errorf("want %s, got %s", path, Wd())
	}

	os.Setenv("SNAP_DATA", "")
	if cwd != Wd() {
		t.Errorf("want %s, got %s", cwd, Wd())
	}

}
