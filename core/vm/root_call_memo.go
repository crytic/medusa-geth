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
	"github.com/crytic/medusa-geth/params"
)

// RootCallMemo is an optional caller-owned exact memo for the root interpreter
// frame. The caller must prove that reuse preserves all state and observer
// effects. Transaction validation and the surrounding EVM call still execute.
// The default nil setting preserves ordinary interpreter behavior.
type RootCallMemo interface {
	// Lookup may return a hit only for a successful execution whose state, gas,
	// return data and observer effects the implementation can exactly reproduce.
	// The EVM does not validate hits. Input and contract data are borrowed.
	Lookup(*EVM, *Contract, []byte, bool, params.Rules) (output, returnData []byte, remainingGas uint64, hit bool)
	// Store observes an executed root frame, including errors. Implementations
	// retaining the borrowed output or return data must make their own copies.
	Store(output, returnData []byte, remainingGas uint64, err error)
}

// IsPrecompile reports whether the current EVM will dispatch an address to a
// precompiled implementation, including Medusa's additional precompiles.
func (evm *EVM) IsPrecompile(address common.Address) bool {
	_, ok := evm.precompile(address)
	return ok
}
