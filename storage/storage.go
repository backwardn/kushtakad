// Copyright 2016-2019 DutchSec (https://dutchsec.com/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package storage

import (
	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/state"
)

type storage interface {
	Get(string) error
	Set(string, []byte) error
}

const bktName = "storage_bkt"

var db *storm.DB
var dataDir string

// SetDataDir
func SetDataDir(s string) {
	var err error
	if db != nil {
		return
	}

	dataDir = s
	db, err = MustDB()
	if err != nil {
		log.Fatal(err)
	}
}

// MustDB
func MustDB() (*storm.DB, error) {
	db, err := storm.Open(state.DbSensorLocation())
	if err != nil {
		log.Fatalf("Failed to open database : %s", err)
		return nil, err
	}

	return db, nil
}

// MustDB
func MustDBWithName(dbname string) (*storm.DB, error) {
	db, err := storm.Open(state.DbLocationWithName(dbname))
	if err != nil {
		log.Fatalf("Failed to open database : %s", err)
		return nil, err
	}

	return db, nil
}

// Storage interface
type Storage interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte) error
}

// Namespace sets the namespace prefix
func Namespace(namespace string) (*stormStorage, error) {
	prefix := make([]byte, len(namespace)+1)

	_ = copy(prefix, namespace)

	prefix[len(namespace)] = byte('.')

	db, err := MustDB()
	if err != nil {
		return nil, err
	}

	return &stormStorage{
		db: db,
		ns: prefix,
	}, nil
}

type stormStorage struct {
	db *storm.DB

	ns []byte
}

func (s *stormStorage) Get(key string) ([]byte, error) {
	log.Debug("stormStorage Get()")

	val := []byte{}

	k := append(s.ns, key...)

	tx, err := s.db.Bolt.Begin(true)
	if err != nil {
		log.Debugf("Unable to begin tx %v", err)
		return val, err
	}

	bkt, err := tx.CreateBucketIfNotExists([]byte(bktName))
	if err != nil {
		log.Debugf("Unable to create bucket using tx %v", err)
		tx.Rollback()
		return val, err
	}

	val = bkt.Get([]byte(k))

	err = tx.Commit()
	if err != nil {
		log.Debugf("Unable to commit tx %v", err)
		tx.Rollback()
		return val, err
	}
	return val, err
}

func (s *stormStorage) Set(key string, data []byte) error {
	log.Debug("stormStorage Begin Set")
	log.Debugf("Set key = %s", key)
	k := append(s.ns, key...)

	tx, err := s.db.Bolt.Begin(true)
	if err != nil {
		log.Debugf("Unable to begin tx %v", err)
		return err
	}

	bkt, err := tx.CreateBucketIfNotExists([]byte(bktName))
	if err != nil {
		log.Debugf("Unable to create bucket using tx %v", err)
		tx.Rollback()
		return err
	}

	err = bkt.Put([]byte(k), data)
	if err != nil {
		log.Debugf("Unable to put data into bucket using tx %v", err)
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Debugf("Unable to commit tx %v", err)
		tx.Rollback()
		return err
	}

	log.Debug("stormStorage End Set")

	return nil
}
