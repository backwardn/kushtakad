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
	tmpDir       = "tmp"
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
	tmpPath          string
)

// SetupFileStructure makes sure the files on the file system are in the correct state
// if they are not, the application must fail
func SetupFileStructure(box *packr.Box) error {
	// setup data dir base on environment
	dataDir = helpers.DataDir()

	imagesPath = path.Join(Wd(), helpers.DataDir(), imagesDir)
	if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
		err = os.MkdirAll(imagesPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", imagesPath))
		}
	}

	sessionsPath = path.Join(Wd(), helpers.DataDir(), sessionsDir)
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		err = os.MkdirAll(sessionsPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", sessionsPath))
		}
	}

	logsPath = path.Join(Wd(), helpers.DataDir(), logsDir)
	if _, err := os.Stat(logsPath); os.IsNotExist(err) {
		err = os.MkdirAll(logsPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", logsPath))
		}
	}

	acmeProdPath = path.Join(Wd(), helpers.DataDir(), acmeDir, acmeProd)
	if _, err := os.Stat(acmeProdPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeProdPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeProdPath))
		}
	}

	acmeTestPath = path.Join(Wd(), helpers.DataDir(), acmeDir, acmeTest)
	if _, err := os.Stat(acmeTestPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeTestPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeTestPath))
		}
	}

	clonesPath = path.Join(Wd(), helpers.DataDir(), clonesDir)
	if _, err := os.Stat(clonesPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesPath))
		}
	}

	clonesSensorPath = path.Join(Wd(), helpers.DataDir(), clonesDir, clonesSensor)
	if _, err := os.Stat(clonesSensorPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesSensorPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesSensorPath))
		}
	}

	clonesServerPath = path.Join(Wd(), helpers.DataDir(), clonesDir, clonesServer)
	if _, err := os.Stat(clonesServerPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesServerPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesServerPath))
		}
	}

	tmpPath = path.Join(Wd(), helpers.DataDir(), tmpDir)
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		err = os.MkdirAll(tmpPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", tmpPath))
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
			err = os.MkdirAll(fullDir, 0744)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", dir))
			}
		}

		fullPath := path.Join(themePath, fpath)
		s, err := b.Find(fpath)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to find file in box %s", fullPath))
		}

		err = ioutil.WriteFile(fullPath, s, 0744)
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
	return path.Join(Wd(), helpers.DataDir(), dbname)
}

func DbWithLocationWithName(location, dbname string) string {
	return path.Join(location, dbname)
}

func ClonesLocation() string {
	return path.Join(Wd(), helpers.DataDir(), clonesDir)
}

func SensorCfgLocation() string {
	return path.Join(Wd(), helpers.DataDir(), sensorCfg)
}

func ServerCfgLocation() string {
	return path.Join(Wd(), helpers.DataDir(), serverCfg)
}

func SensorClonesLocation() string {
	return path.Join(Wd(), helpers.DataDir(), clonesDir, clonesSensor)
}

func LogsLocation() string {
	return path.Join(Wd(), helpers.DataDir(), logsDir)
}

func TmpDirLocation() string {
	return path.Join(Wd(), helpers.DataDir(), tmpDir)
}

func ServerClonesLocation() string {
	return path.Join(Wd(), helpers.DataDir(), clonesDir, clonesServer)
}

func SessionLocation() string {
	return path.Join(Wd(), helpers.DataDir(), sessionsDir)
}

func AcmeProdLocation() string {
	return path.Join(Wd(), helpers.DataDir(), acmeDir, acmeProd)
}

func AcmeTestLocation() string {
	return path.Join(Wd(), helpers.DataDir(), acmeDir, acmeTest)
}

func DataDirLocation() string {

	return path.Join(Wd(), helpers.DataDir())
}

// Wd returns the acting current working directory
// this path can change for certain configs
func Wd() string {
	if len(os.Getenv("SNAP_DATA")) > 1 {
		return path.Join(os.Getenv("SNAP_DATA"))
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Cwd () unable to detect current working directory"))
	}

	return cwd

}
