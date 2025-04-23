package arbitrum

import (
	"context"

	"github.com/crytic/medusa-geth/arbitrum_types"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/types"
)

type ArbInterface interface {
	PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error
	BlockChain() *core.BlockChain
	ArbNode() interface{}
}
