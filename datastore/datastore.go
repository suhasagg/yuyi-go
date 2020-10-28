// Copyright 2015 The yuyi Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datastore

import (
	"sync"
	"time"
)

var (
	minSeq = uint64(0)
	maxSeq = uint64(0xFFFFFFFFFFFFFFFF)

	// register all active datastore
	stores = make([]*DataStore, 0)

	done   = make(chan bool)
	ticker = time.NewTicker(5 * time.Second)
)

func init() {
	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			for _, store := range stores {
				// check if the store should do flushing
				store.checkForFlushing()
			}
		}
	}
}

type DataStore struct {
	// name the name of the datastore
	name string

	// activeMemTable current using memory table for writing data to.
	activeMemTable MemTable

	// sealedMemTables the memory table instances that already reached
	//max limit size and need be persisted
	sealedMemTables []*MemTable

	// btree the structure for persisting data from memory table
	btree *BTree

	// seq the sequence of write operation. When read started, current
	// sequence will be acquired and make sure that no later committed
	// kv will be read
	seq uint64

	// mu the lock for checking if memory table is ready to dump
	mu sync.Mutex
}

func (store *DataStore) Put(key Key, value Value) {
	return
}

func (store *DataStore) Remove(key Key) {
	return
}

func (store *DataStore) Has(key Key) bool {
	seq := store.seq
	value := store.activeMemTable.Get(key, seq)
	if value != nil {
		if value.Operation == Remove {
			return false
		}
		return true
	}
	for _, table := range store.sealedMemTables {
		value := table.Get(key, maxSeq)
		if value != nil {
			if value.Operation == Remove {
				return false
			}
			return true
		}
	}
	return store.btree.Has(&key)
}

func (store *DataStore) Get(key Key) Value {
	seq := store.seq
	value := store.activeMemTable.Get(key, seq)
	if value != nil {
		if value.Operation == Remove {
			return nil
		}
		return value.Value
	}
	for _, table := range store.sealedMemTables {
		value := table.Get(key, maxSeq)
		if value != nil {
			if value.Operation == Remove {
				return nil
			}
			return value.Value
		}
	}
	return store.btree.Get(&key)
}

func (store *DataStore) List(start Key, end Key, max int) []*KVPair {
	// seq := store.seq
	// activeIter := store.activeMemTable.List(start, end, seq)
	// sealedIter := make([]*memtable.Iterator, 0)
	// for _, memtable := range store.sealedMemTables {
	// 	sealedIter = append(sealedIter, memtable.List(start, end, seq))
	// }
	// resFromTree := store.btree.List(start, end, max)

	// res := make([]*memtable.KVPair, 0)
	// found := 0
	// Todo: implement min heap for merging result together
	return nil
}

func (store *DataStore) ReverseList(start Key, end Value, max uint16) []Value {
	return nil
}

func (store *DataStore) checkForFlushing() {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.shouldFlush() {
		go store.flush()
	}
}

func (store *DataStore) shouldFlush() bool {
	return len(store.sealedMemTables) != 0 && store.btree.isDumping()
}

func (store *DataStore) flush() {
	sealed := store.sealedMemTables[0:]

	dumper := newDumper(store.btree)
	store.btree.lastTreeInfo = dumper.Dump(mergeMemTables(sealed))
}

func mergeMemTables(tables []*MemTable) []*KVEntry {
	return nil
}
