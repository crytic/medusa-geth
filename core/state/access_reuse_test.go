// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/binary"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/params"
)

func TestAccessListReuse(t *testing.T) {
	a, b := common.Address{1}, common.Address{2}
	x, y := common.Hash{1}, common.Hash{2}
	al := newAccessList()
	al.AddSlot(a, x)
	copied := al.Copy()
	al.DeleteSlot(a, x)
	al.DeleteAddress(a)
	al.AddSlot(b, y)
	if _, present := al.Contains(b, x); present {
		t.Fatal("reused slot map retained reverted slot")
	}
	if _, present := copied.Contains(a, x); !present {
		t.Fatal("copy was aliased")
	}
	al.reset()
	if al.ContainsAddress(b) {
		t.Fatal("warm address leaked across transactions")
	}
	al.AddSlot(a, x)
	if _, present := al.Contains(a, y); present {
		t.Fatal("warm slot leaked across transactions")
	}
	if _, present := al.Contains(a, x); !present {
		t.Fatal("new slot missing")
	}
	al.DeleteSlot(a, x)
	al.DeleteAddress(a)
	if al.ContainsAddress(a) {
		t.Fatal("rollback failed after reuse")
	}
}

func TestAccessListReuseAfterLargeRollback(t *testing.T) {
	for _, slots := range []bool{false, true} {
		al := newAccessList()
		address := common.Address{1}
		for i := uint64(0); i < 4097; i++ {
			if slots {
				var key common.Hash
				binary.BigEndian.PutUint64(key[:8], i)
				al.AddSlot(address, key)
			} else {
				var address common.Address
				binary.BigEndian.PutUint64(address[:8], i)
				al.AddAddress(address)
			}
		}
		// Rollback must not hide an exceptional allocation from the next reset.
		for i := uint64(0); i < 4097; i++ {
			if slots {
				var key common.Hash
				binary.BigEndian.PutUint64(key[:8], i)
				al.DeleteSlot(address, key)
			} else {
				var address common.Address
				binary.BigEndian.PutUint64(address[:8], i)
				al.DeleteAddress(address)
			}
		}
		if !al.oversized {
			t.Fatal("large allocation was not recorded")
		}
		copied := al.Copy()
		for _, list := range []*accessList{al, copied} {
			list.reset()
			if list.oversized || len(list.addresses) != 0 || list.slots != nil {
				t.Fatal("large allocation retained after reset")
			}
		}
	}
}

func TestAccessListReuseManySlotMaps(t *testing.T) {
	al := newAccessList()
	for i := uint64(0); i < 257; i++ {
		var address common.Address
		binary.BigEndian.PutUint64(address[:8], i)
		al.AddSlot(address, common.Hash{1})
	}
	al.reset()
	if al.slots != nil || len(al.addresses) != 0 {
		t.Fatal("large slot-map array retained")
	}
}

func TestStateDBAccessListReuse(t *testing.T) {
	s, err := New(types.EmptyRootHash, NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	sender, coinbase, dst, precompile, listed, old := common.Address{1}, common.Address{2}, common.Address{3}, common.Address{4}, common.Address{5}, common.Address{6}
	slot := common.Hash{1}
	s.AddSlotToAccessList(old, slot)
	copied := s.Copy()
	s.Prepare(params.Rules{IsEIP2929: true, IsShanghai: true}, sender, coinbase, &dst, []common.Address{precompile}, types.AccessList{{Address: listed, StorageKeys: []common.Hash{slot}}})
	if s.AddressInAccessList(old) {
		t.Fatal("previous transaction address stayed warm")
	}
	for _, addr := range []common.Address{sender, coinbase, dst, precompile, listed} {
		if !s.AddressInAccessList(addr) {
			t.Fatalf("prepared address %s is cold", addr)
		}
	}
	if _, present := s.SlotInAccessList(listed, slot); !present {
		t.Fatal("declared access-list slot is cold")
	}
	if _, present := copied.SlotInAccessList(old, slot); !present {
		t.Fatal("reset modified copied state")
	}
	snapshot := s.Snapshot()
	s.AddSlotToAccessList(old, slot)
	s.RevertToSnapshot(snapshot)
	if s.AddressInAccessList(old) {
		t.Fatal("rollback after reuse left address warm")
	}
	s.Prepare(params.Rules{IsEIP2929: true}, sender, coinbase, nil, nil, nil)
	if s.AddressInAccessList(coinbase) || s.AddressInAccessList(dst) || s.AddressInAccessList(listed) {
		t.Fatal("new transaction inherited old access-list warmth")
	}
}
