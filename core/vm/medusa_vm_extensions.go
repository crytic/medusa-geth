package vm

import "github.com/ethereum/go-ethereum/common"

// MEDUSA: This entire file defines extensions for this package, used to power medusa features.

// ConfigExtensions defines extended properties to be inherited by the Config.
// Note: Ensure any values which are added here and were not set do not change default EVM behaviour.
type ConfigExtensions struct {
	// OverrideCodeSizeCheck indicates whether code size checks should be disabled.
	OverrideCodeSizeCheck bool

	// AdditionalPrecompiles defines additional precompile contracts to be used by the VM.
	AdditionalPrecompiles map[common.Address]PrecompiledContract
}
