package version

import (
	"os"

	"github.com/kushtaka/kushtakad/helpers"
)

var (
	current string
)

func Current() string {
	return current
}

func Sanity() {
	env := os.Getenv("KUSHTAKA_ENV")
	if env == helpers.StateTest || env == helpers.StateDevelopment {
		if len(current) != 0 {
			panic("the version information should be zero during testing and development")
		}
	}

	if len(current) <= 5 {
		panic("the version information appears missing during compilation : `ldflags -X`")
	}
}
