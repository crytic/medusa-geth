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
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/common/lru"
)

// Analysis is immutable and content-addressed. Keep transaction-local and
// contract-local caches in front of this shared cache, so locks are taken only
// on the first jump in each distinct code hash per EVM. Sharding avoids a single
// lock across workers. Bound both entries and individual bitmap sizes: at most
// 2048 entries and roughly 144 MiB of code and bitmap storage, plus bookkeeping.
// Retain code bytes as well as their hash so externally supplied hashes cannot
// cause one EVM to reuse the analysis of different code from another EVM.
const maxSharedAnalysisCodeSize = 64 * 1024

// jumpdestAnalysis owns immutable copies of the code and its PUSH-data bitmap.
type jumpdestAnalysis struct {
	code   string
	bitmap bitvec
}

var sharedJumpdestCache = func() [16]*lru.Cache[common.Hash, jumpdestAnalysis] {
	var shards [16]*lru.Cache[common.Hash, jumpdestAnalysis]
	for i := range shards {
		shards[i] = lru.NewCache[common.Hash, jumpdestAnalysis](128)
	}
	return shards
}()

// cachedCodeBitmap reuses analysis only when both the hash and code match.
func cachedCodeBitmap(hash common.Hash, code []byte) bitvec {
	if len(code) > maxSharedAnalysisCodeSize {
		return codeBitmap(code)
	}
	cache := sharedJumpdestCache[hash[0]&15]
	if analysis, ok := cache.Get(hash); ok && analysis.code == string(code) {
		return analysis.bitmap
	}
	analysis := codeBitmap(code)
	cache.Add(hash, jumpdestAnalysis{code: string(code), bitmap: analysis})
	return analysis
}
