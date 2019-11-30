package state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/gobuffalo/packr/v2/file"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/pkg/errors"
)

const (
	staticDir    = "static"
	imagesDir    = "images"
	sessionsDir  = "sessions"
	logsDir      = "logs"
	clonesDir    = "clones"
	clonesServer = "server"
	clonesSensor = "sensor"
	acmeDir      = "acme"
	acmeProd     = "prod"
	acmeTest     = "test"
	dbFile       = "kushtaka.db"
	dbSensor     = "sensor.db"
	serverCfg    = "server.json"
	sensorCfg    = "sensor.json"
)

var (
	dataDir          string
	cwd              string
	themePath        string
	imagesPath       string
	sessionsPath     string
	acmeProdPath     string
	acmeTestPath     string
	clonesPath       string
	clonesServerPath string
	clonesSensorPath string
	logsPath         string
	sensorCfgPath    string
)

// SetupFileStructure makes sure the files on the file system are in the correct state
// if they are not, the application must fail
func SetupFileStructure(box *packr.Box) error {
	var err error

	// setup data dir base on environment
	dataDir = helpers.DataDir()

	cwd, err = os.Getwd()
	if err != nil {
		return errors.Wrap(err, "unable to detect current working directory")
	}

	imagesPath = path.Join(cwd, helpers.DataDir(), imagesDir)
	if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
		err = os.MkdirAll(imagesPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", imagesPath))
		}
	}

	sessionsPath = path.Join(cwd, helpers.DataDir(), sessionsDir)
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		err = os.MkdirAll(sessionsPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", sessionsPath))
		}
	}

	logsPath = path.Join(cwd, helpers.DataDir(), logsDir)
	if _, err := os.Stat(logsPath); os.IsNotExist(err) {
		err = os.MkdirAll(logsPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", logsPath))
		}
	}

	acmeProdPath = path.Join(cwd, helpers.DataDir(), acmeDir, acmeProd)
	if _, err := os.Stat(acmeProdPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeProdPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeProdPath))
		}
	}

	acmeTestPath = path.Join(cwd, helpers.DataDir(), acmeDir, acmeTest)
	if _, err := os.Stat(acmeTestPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeTestPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeTestPath))
		}
	}

	clonesPath = path.Join(cwd, helpers.DataDir(), clonesDir)
	if _, err := os.Stat(clonesPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesPath))
		}
	}

	clonesSensorPath = path.Join(cwd, helpers.DataDir(), clonesDir, clonesSensor)
	if _, err := os.Stat(clonesSensorPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesSensorPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesSensorPath))
		}
	}

	clonesServerPath = path.Join(cwd, helpers.DataDir(), clonesDir, clonesServer)
	if _, err := os.Stat(clonesServerPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesServerPath, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesServerPath))
		}
	}

	return nil
}

// createFiles is an abstraction that actually creates the directories and files
// it walks the packr box looking for the base files to write to the file system
func createFiles(b *packr.Box) error {
	err := b.Walk(func(fpath string, f file.File) error {
		dir, _ := path.Split(fpath)
		fullDir := path.Join(themePath, dir)

		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			err = os.MkdirAll(fullDir, 0644)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", dir))
			}
		}

		fullPath := path.Join(themePath, fpath)
		s, err := b.Find(fpath)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to find file in box %s", fullPath))
		}

		err = ioutil.WriteFile(fullPath, s, 0644)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to create file %s", fullPath))
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func DbLocation() string {
	return DbLocationWithName(dbFile)
}

func DbSensorLocation() string {
	return DbLocationWithName(dbSensor)
}

func DbLocationWithName(dbname string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "DbLocationWithName() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), dbname)
}

func DbWithLocationWithName(location, dbname string) string {
	return path.Join(location, dbname)
}

func ClonesLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "ClonesLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), clonesDir)
}

func SensorCfgLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "SensorCfgLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), sensorCfg)
}

func ServerCfgLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "ServerCfgLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), serverCfg)
}

func SensorClonesLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "SensorClonesLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), clonesDir, clonesSensor)
}

func LogsLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "LogsLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), logsDir)
}

func ServerClonesLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "ClonesLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), clonesDir, clonesServer)
}

func SessionLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "SessionLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), sessionsDir)
}

func AcmeProdLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "AcmeProdLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), acmeDir, acmeProd)
}

func AcmeTestLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "AcmeTestLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir(), acmeDir, acmeTest)
}

func DataDirLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "helpers.DataDir()Location() unable to detect current working directory"))
	}

	return path.Join(cwd, helpers.DataDir())
}
