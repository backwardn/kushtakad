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
	"log"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/state"
	"go.etcd.io/bbolt"
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
	if db != nil {
		return
	}

	dataDir = s
	db = MustDB()
}

// MustDB
func MustDB() *storm.DB {
	db, err := storm.Open(state.DbSensorLocation())
	if err != nil {
		log.Fatalf("Failed to open database : %s", err)
		return nil
	}

	return db
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

	return &stormStorage{
		db: MustDB(),
		ns: prefix,
	}, nil
}

type stormStorage struct {
	db *storm.DB

	ns []byte
}

func (s *stormStorage) Get(key string) ([]byte, error) {

	val := []byte{}

	k := append(s.ns, key...)

	err := s.db.Bolt.View(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bktName))
		if err != nil {
			return err
		}
		val = bucket.Get([]byte(k))
		return nil
	})

	return val, err
}

func (s *stormStorage) Set(key string, data []byte) error {
	k := append(s.ns, key...)

	err := s.db.Bolt.Update(func(tx *bbolt.Tx) error {
		dbx := s.db.WithTransaction(tx)
		err := dbx.Set(bktName, k, data)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
