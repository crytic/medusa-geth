package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type RemoteStateProvider interface {
	// ImportStorageAt Provides the value of the specified storage slot on the forked network. This can only be called
	// once for each (addr,slot) pair for any series of un-reverted snapshots. If it's called twice, and the original
	// snapshot it was called during was not reverted, it will return a RemoteStorageError error.
	ImportStorageAt(addr common.Address, slot common.Hash, snapId int) (data common.Hash, err *RemoteStorageError)

	// ImportStateObject Provides the various top-level values of the state object on the forked network. This can only
	// be caled once for each addr for any series of un-reverted snapshots. If it's called twice, and the original
	// snapshot it was called during was not reverted, it will return a RemoteStateError error.
	ImportStateObject(addr common.Address, snapId int) (bal *uint256.Int, nonce uint64, code []byte, e *RemoteStateError)

	// MarkSlotWritten Notifies the state provider that there has been a write to the specified slot, and unless this
	// snapshot reverts, future imports for this slot should be forbidden.
	MarkSlotWritten(addr common.Address, slot common.Hash, snapId int)

	// MarkStateObjectWritten Notifies the state provider that there has been a write to the specified account, and
	// unless this snapshot reverts, future imports for this account should be forbidden.
	// MarkStateObjectWritten(addr common.Address, snapId int)

	// NotifyRevertedToSnapshot Notifies the state provider that the data it imported beyond snapId has been reverted.
	NotifyRevertedToSnapshot(snapId int)
}

type RemoteStateProviderFactory interface {
	New() RemoteStateProvider
}

type RemoteStateError struct {
	CannotQueryDirtyAccount bool
	Error                   error
}

type RemoteStorageError struct {
	CannotQueryDirtySlot bool
	Error                error
}
