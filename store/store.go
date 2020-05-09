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

type Store struct {
	db *LevelDBStore
}

func NewStore(path string) (*Store, error) {
	ldb, err := NewLevelDBStore(path)
	if err != nil {
		return nil, err
	}
	st := &Store{db: ldb}
	return st, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) SaveData(key, value []byte) error {
	return s.db.Put(key, value)
}

func (s *Store) GetData(key []byte) ([]byte, error) {
	return s.db.Get(key)
}

func (s *Store) NewBatch() {
	s.db.NewBatch()
}

func (s *Store) BatchPut(key, value []byte) {
	s.db.BatchPut(key, value)
}

func (s *Store) CommitTo() error {
	return s.db.BatchCommit()
}
