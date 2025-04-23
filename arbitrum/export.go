package arbitrum

import (
	"context"

	"github.com/crytic/medusa-geth/common/hexutil"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/internal/ethapi"
	"github.com/crytic/medusa-geth/internal/ethapi/override"
	"github.com/crytic/medusa-geth/rpc"
)

type TransactionArgs = ethapi.TransactionArgs

func EstimateGas(ctx context.Context, b ethapi.Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, gasCap uint64) (hexutil.Uint64, error) {
	return ethapi.DoEstimateGas(ctx, b, args, blockNrOrHash, overrides, blockOverrides, gasCap)
}

func NewRevertReason(result *core.ExecutionResult) error {
	return ethapi.NewRevertError(result)
}
