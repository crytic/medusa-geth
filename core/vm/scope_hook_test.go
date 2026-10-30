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
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/params"
	"github.com/holiman/uint256"
)

func TestScopeOpcodeHook(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		scopes, opcodes, faults := 0, 0, 0
		hooks := &tracing.Hooks{
			OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
				t.Error("fallback opcode hook called")
			},
			OnOpcodeForScope: func(scope tracing.OpContext) tracing.OpcodeHook {
				scopes++
				if scope.(*ScopeContext).Contract == nil {
					t.Fatal("missing contract")
				}
				if disabled {
					return nil
				}
				return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { opcodes++ }
			},
			OnFault: func(uint64, byte, uint64, uint64, tracing.OpContext, int, error) { faults++ },
		}
		evm := NewEVM(BlockContext{BlockNumber: big.NewInt(0)}, nil, params.TestChainConfig, Config{Tracer: hooks})
		c := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 1000, nil)
		c.SetCallCode(common.Hash{}, []byte{byte(PUSH1), 1, byte(INVALID)})
		if _, err := evm.Interpreter().Run(c, nil, false); err == nil {
			t.Fatal("expected invalid opcode error")
		}
		if scopes != 1 || faults != 1 {
			t.Fatalf("scopes=%d faults=%d", scopes, faults)
		}
		if disabled && opcodes != 0 || !disabled && opcodes != 2 {
			t.Fatalf("opcodes=%d disabled=%v", opcodes, disabled)
		}
	}
}

func TestLegacyOpcodeHookMutation(t *testing.T) {
	first, replaced := 0, 0
	hooks := &tracing.Hooks{}
	hooks.OnOpcode = func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
		first++
		hooks.OnOpcode = func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { replaced++ }
	}
	evm := NewEVM(BlockContext{BlockNumber: big.NewInt(0)}, nil, params.TestChainConfig, Config{Tracer: hooks})
	contract := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 1000, nil)
	contract.SetCallCode(common.Hash{}, []byte{byte(PUSH1), 1, byte(PUSH1), 2, byte(ADD), byte(STOP)})
	if _, err := evm.Interpreter().Run(contract, nil, false); err != nil {
		t.Fatal(err)
	}
	if first != 1 || replaced != 3 {
		t.Fatalf("legacy callbacks first=%d replaced=%d", first, replaced)
	}
}

func TestScopeOpcodeHookNested(t *testing.T) {
	selected, parentCalls, childCalls := 0, 0, 0
	hooks := &tracing.Hooks{}
	evm := NewEVM(BlockContext{BlockNumber: big.NewInt(0)}, nil, params.TestChainConfig, Config{Tracer: hooks})
	child := NewContract(common.Address{}, common.Address{1}, new(uint256.Int), 1000, nil)
	child.SetCallCode(common.Hash{}, []byte{byte(STOP)})
	hooks.OnOpcodeForScope = func(scope tracing.OpContext) tracing.OpcodeHook {
		selected++
		if scope.Address() == (common.Address{1}) {
			return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { childCalls++ }
		}
		return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
			parentCalls++
			if parentCalls == 1 {
				if _, err := evm.Interpreter().Run(child, nil, false); err != nil {
					t.Error(err)
				}
			}
		}
	}
	parent := NewContract(common.Address{}, common.Address{}, new(uint256.Int), 1000, nil)
	parent.SetCallCode(common.Hash{}, []byte{byte(PUSH1), 0, byte(STOP)})
	if _, err := evm.Interpreter().Run(parent, nil, false); err != nil {
		t.Fatal(err)
	}
	if selected != 2 || parentCalls != 2 || childCalls != 1 {
		t.Fatalf("scopes=%d parent=%d child=%d", selected, parentCalls, childCalls)
	}
}
