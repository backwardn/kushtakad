package helpers

import "os"

const (
	StateProduction  = "production"
	StateTest        = "test"
	StateDevelopment = "development"
	prodDataDir      = "data"
	testDataDir      = "data_test"
	devDataDir       = "data_dev"
)

func DataDir() string {
	env := os.Getenv("KUSHTAKA_ENV")
	if env == StateDevelopment {
		return devDataDir
	} else if env == StateTest {
		return testDataDir
	}
	return prodDataDir
}
