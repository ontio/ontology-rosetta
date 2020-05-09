/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */
package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type LevelDBStore struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

func NewLevelDBStore(path string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}
	if err != nil {
		return nil, err
	}
	return &LevelDBStore{
		db:    db,
		batch: nil,
	}, nil
}

//Put a key-value pair to leveldb
func (self *LevelDBStore) Put(key []byte, value []byte) error {
	return self.db.Put(key, value, nil)
}

//Get the value of a key from leveldb
func (self *LevelDBStore) Get(key []byte) ([]byte, error) {
	dat, err := self.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

//Has return whether the key is exist in leveldb
func (self *LevelDBStore) Has(key []byte) (bool, error) {
	return self.db.Has(key, nil)
}

//Delete the the in leveldb
func (self *LevelDBStore) Delete(key []byte) error {
	return self.db.Delete(key, nil)
}

//Close leveldb
func (self *LevelDBStore) Close() error {
	err := self.db.Close()
	return err
}

//NewBatch start commit batch
func (self *LevelDBStore) NewBatch() {
	self.batch = new(leveldb.Batch)
}

//BatchPut put a key-value pair to leveldb batch
func (self *LevelDBStore) BatchPut(key []byte, value []byte) {
	self.batch.Put(key, value)
}

//BatchDelete delete a key to leveldb batch
func (self *LevelDBStore) BatchDelete(key []byte) {
	self.batch.Delete(key)
}

//BatchCommit commit batch to leveldb
func (self *LevelDBStore) BatchCommit() error {
	err := self.db.Write(self.batch, nil)
	if err != nil {
		return err
	}
	self.batch = nil
	return nil
}
