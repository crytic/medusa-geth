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

package vm

import (
	"sync"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/crypto"
	"github.com/holiman/uint256"
)

func TestSharedJumpdestAnalysis(t *testing.T) {
	code := []byte{byte(PUSH1), byte(JUMPDEST), byte(JUMPDEST), byte(STOP)}
	hash := crypto.Keccak256Hash(code)
	var first bitvec
	for i := 0; i < 2; i++ {
		c := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
		c.SetCallCode(hash, code)
		if c.validJumpdest(uint256.NewInt(1)) || !c.validJumpdest(uint256.NewInt(2)) {
			t.Fatal("incorrect PUSH-data analysis")
		}
		if i == 0 {
			first = c.analysis
		} else if &first[0] != &c.analysis[0] {
			t.Fatal("analysis not reused across contracts/EVMs")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if got := cachedCodeBitmap(hash, code); &got[0] != &first[0] {
					t.Error("concurrent cache miss")
				}
			}
		}()
	}
	wg.Wait()
	// A code change uses a new hash, even if deployed at the same address.
	changed := []byte{byte(JUMPDEST), byte(JUMPDEST), byte(STOP)}
	other := cachedCodeBitmap(crypto.Keccak256Hash(changed), changed)
	if !other.codeSegment(1) {
		t.Fatal("stale analysis after code change")
	}
	// Large code still works with code-size checks disabled, but is not retained.
	large := make([]byte, maxSharedAnalysisCodeSize+1)
	largeHash := crypto.Keccak256Hash(large)
	cachedCodeBitmap(largeHash, large)
	if _, ok := sharedJumpdestCache[largeHash[0]&15].Get(largeHash); ok {
		t.Fatal("oversized entry retained")
	}
}

// Caller-supplied hashes must not allow cross-EVM cache pollution.
func TestSharedJumpdestDifferentCode(t *testing.T) {
	hash := common.Hash{0xff, 0x42}
	code := []byte{byte(PUSH1), byte(JUMPDEST), byte(JUMPDEST)}
	makeContract := func() *Contract {
		c := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
		c.SetCallCode(hash, code)
		return c
	}
	if makeContract().validJumpdest(uint256.NewInt(1)) {
		t.Fatal("PUSH data treated as code")
	}
	code[0] = byte(STOP)
	if !makeContract().validJumpdest(uint256.NewInt(1)) {
		t.Fatal("analysis reused for different code with the same hash")
	}
	code = append(code, byte(JUMPDEST))
	if !makeContract().validJumpdest(uint256.NewInt(3)) {
		t.Fatal("analysis reused after code length changed")
	}
}

func TestSharedJumpdestInitcode(t *testing.T) {
	c := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
	c.SetCallCode(common.Hash{}, []byte{byte(PUSH1), byte(JUMPDEST), byte(JUMPDEST)})
	if c.validJumpdest(uint256.NewInt(1)) || !c.validJumpdest(uint256.NewInt(2)) {
		t.Fatal("incorrect initcode analysis")
	}
	if len(c.jumpdests) != 0 {
		t.Fatal("initcode stored in parent cache")
	}
}

func TestSharedJumpdestBound(t *testing.T) {
	// Fill one shard past its limit, using distinct caller-supplied hashes.
	for i := 0; i < 256; i++ {
		cachedCodeBitmap(common.Hash{0x0e, byte(i), 0xa7}, []byte{byte(JUMPDEST)})
	}
	if got := len(sharedJumpdestCache[14].Keys()); got > 128 {
		t.Fatalf("cache shard retained %d entries, want at most 128", got)
	}
}
