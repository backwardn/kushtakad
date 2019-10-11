package state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/gobuffalo/packr/v2/file"
	"github.com/pkg/errors"
)

const (
	staticDir   = "static"
	dataDir     = "data"
	imagesDir   = "images"
	sessionsDir = "sessions"
	clonesDir   = "clones"
	acmeDir     = "acme"
	acmeProd    = "prod"
	acmeTest    = "test"
	dbFile      = "kushtaka.db"
	dbSensor    = "sensor.db"
)

var cwd, themePath, imagesPath, sessionsPath, acmeProdPath, acmeTestPath, clonesPath string

// SetupFileStructure makes sure the files on the file system are in the correct state
// if they are not, the application must fail
func SetupFileStructure(box *packr.Box) error {
	var err error

	cwd, err = os.Getwd()
	if err != nil {
		return errors.Wrap(err, "unable to detect current working directory")
	}

	imagesPath = path.Join(cwd, dataDir, imagesDir)
	if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
		err = os.MkdirAll(imagesPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", imagesPath))
		}
	}

	sessionsPath = path.Join(cwd, dataDir, sessionsDir)
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		err = os.MkdirAll(sessionsPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", sessionsPath))
		}
	}

	acmeProdPath = path.Join(cwd, dataDir, acmeDir, acmeProd)
	if _, err := os.Stat(acmeProdPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeProdPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeProdPath))
		}
	}

	acmeTestPath = path.Join(cwd, dataDir, acmeDir, acmeTest)
	if _, err := os.Stat(acmeTestPath); os.IsNotExist(err) {
		err = os.MkdirAll(acmeTestPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", acmeTestPath))
		}
	}

	clonesPath = path.Join(cwd, dataDir, clonesDir)
	if _, err := os.Stat(clonesPath); os.IsNotExist(err) {
		err = os.MkdirAll(clonesPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", clonesPath))
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

	return path.Join(cwd, dataDir, dbname)
}

func DbWithLocationWithName(location, dbname string) string {
	return path.Join(location, dbname)
}

func ClonesLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "ClonesLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, dataDir, clonesDir)
}

func SessionLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "SessionLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, dataDir, sessionsDir)
}

func AcmeProdLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "AcmeProdLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, dataDir, acmeDir, acmeProd)
}

func AcmeTestLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "AcmeTestLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, dataDir, acmeDir, acmeTest)
}

func DataDirLocation() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "DataDirLocation() unable to detect current working directory"))
	}

	return path.Join(cwd, dataDir)
}
