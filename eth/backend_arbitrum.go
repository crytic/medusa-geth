package eth

import (
	"context"

	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/state"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/crytic/medusa-geth/ethdb"
)

func NewArbEthereum(
	blockchain *core.BlockChain,
	chainDb ethdb.Database,
) *Ethereum {
	return &Ethereum{
		blockchain: blockchain,
		chainDb:    chainDb,
	}
}

func (eth *Ethereum) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (*types.Transaction, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return eth.stateAtTransaction(ctx, block, txIndex, reexec)
}
