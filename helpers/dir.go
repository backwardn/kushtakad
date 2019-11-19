package helpers

import "os"

const (
	StateProduction  = "production"
	StateTest        = "test"
	StateDevelopment = "development"
	ProdDataDir      = "data"
	TestDataDir      = "data_test"
	DevDataDir       = "data_dev"
)

func DataDir() string {
	env := os.Getenv("KUSHTAKA_ENV")
	if env == StateDevelopment {
		return DevDataDir
	} else if env == StateTest {
		return TestDataDir
	}
	return ProdDataDir
}
