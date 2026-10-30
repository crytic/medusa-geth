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
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/params"
	"github.com/holiman/uint256"
)

type testRootCallMemo struct {
	lookups, stores int
	hit             bool
	output, data    []byte
	gas             uint64
	err             error
	readOnly        bool
}

func (m *testRootCallMemo) Lookup(_ *EVM, _ *Contract, _ []byte, readOnly bool, _ params.Rules) ([]byte, []byte, uint64, bool) {
	m.lookups++
	m.readOnly = readOnly
	return m.output, m.data, m.gas, m.hit
}
func (m *testRootCallMemo) Store(output, data []byte, gas uint64, err error) {
	m.stores++
	m.output, m.data, m.gas, m.err = bytes.Clone(output), bytes.Clone(data), gas, err
}

func TestRootCallMemo(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(map[bool]string{false: "miss", true: "hit"}[hit], func(t *testing.T) {
			memo := &testRootCallMemo{hit: hit, output: []byte{42}, data: []byte{43}, gas: 91}
			evm := NewEVM(BlockContext{BlockNumber: big.NewInt(1)}, nil, params.AllDevChainProtocolChanges, Config{ReadOnlyCallMemo: memo})
			contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
			contract.SetCallCode(common.Hash{}, []byte{byte(PUSH1), 0, byte(STOP)})
			output, err := evm.Interpreter().Run(contract, []byte{1}, true)
			if err != nil {
				t.Fatal(err)
			}
			if memo.lookups != 1 || !memo.readOnly || evm.depth != 0 || evm.Interpreter().readOnly {
				t.Fatal("incorrect root lifecycle")
			}
			if hit {
				if memo.stores != 0 || !bytes.Equal(output, []byte{42}) || !bytes.Equal(evm.Interpreter().returnData, []byte{43}) || contract.Gas != 91 {
					t.Fatal("hit did not restore result")
				}
			} else if memo.stores != 1 || len(output) != 0 || contract.Gas != 97 || memo.gas != 97 || memo.err != nil {
				t.Fatal("miss did not execute and record")
			}
		})
	}
}

func TestRootCallMemoNestedFrame(t *testing.T) {
	memo := &testRootCallMemo{hit: true}
	evm := NewEVM(BlockContext{BlockNumber: big.NewInt(1)}, nil, params.AllDevChainProtocolChanges, Config{ReadOnlyCallMemo: memo})
	evm.depth = 1
	contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
	contract.SetCallCode(common.Hash{}, []byte{byte(PUSH1), 0, byte(STOP)})
	if _, err := evm.Interpreter().Run(contract, nil, false); err != nil {
		t.Fatal(err)
	}
	if memo.lookups != 0 || memo.stores != 0 || contract.Gas != 97 || evm.depth != 1 {
		t.Fatal("memo intercepted nested frame")
	}
}

func TestRootCallMemoFailure(t *testing.T) {
	memo := &testRootCallMemo{}
	evm := NewEVM(BlockContext{BlockNumber: big.NewInt(1)}, nil, params.AllDevChainProtocolChanges, Config{ReadOnlyCallMemo: memo})
	contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)
	contract.SetCallCode(common.Hash{}, []byte{byte(REVERT)})
	_, err := evm.Interpreter().Run(contract, nil, false)
	if err == nil || memo.stores != 1 || !errors.Is(memo.err, err) {
		t.Fatal("execution failure not recorded")
	}
}
