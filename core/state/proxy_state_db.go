package state

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"sort"
)

var _ vm.StateDB = (*ProxyStateDB)(nil)

type

type ProxyStateDB struct {
	*StateDB
	forkStateProvider ForkedStateProvider
}

func NewProxyDB(root common.Hash, db Database, forkStateProvider ForkedStateProvider) (*ProxyStateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	sdb := &ProxyStateDB{
		forkStateProvider: forkStateProvider,
		StateDB: &StateDB{
			db:                   db,
			trie:                 tr,
			originalRoot:         root,
			snaps:                nil,
			stateObjects:         make(map[common.Address]*stateObject),
			stateObjectsDestruct: make(map[common.Address]*stateObject),
			mutations:            make(map[common.Address]*mutation),
			logs:                 make(map[common.Hash][]*types.Log),
			preimages:            make(map[common.Hash][]byte),
			journal:              newJournal(),
			accessList:           newAccessList(),
			transientStorage:     newTransientStorage(),
			hasher:               crypto.NewKeccakState(),
		},
	}
	return sdb, nil
}

// must be shared across entire set of transactions tested. we'll need an initial one for
// deploying contracts, then a way to clone it for workers
type ForkedStateProvider interface {
	// ImportStorageAt Provides the value of the specified storage slot on the forked network. This can only be called
	// once for each (addr,slot) pair for any series of un-reverted snapshots. If it's called twice, and the original
	// snapshot it was called during was not reverted, it will return a ForkedStorageError error.
	ImportStorageAt(addr common.Address, slot common.Hash, snapId int) (data common.Hash, err *ForkedStorageError)

	// ImportStateObject Provides the various top-level values of the state object on the forked network. This can only
	// be caled once for each addr for any series of un-reverted snapshots. If it's called twice, and the original
	// snapshot it was called during was not reverted, it will return a ForkedStateError error.
	ImportStateObject(addr common.Address, snapId int) (bal *uint256.Int, nonce uint64, code []byte, e *ForkedStateError)

	// MarkSlotWritten Notifies the state provider that there has been a write to the specified slot, and unless this
	// snapshot reverts, future imports for this slot should be forbidden.
	MarkSlotWritten(addr common.Address, slot common.Hash, snapId int)

	// MarkStateObjectWritten Notifies the state provider that there has been a write to the specified account, and
	// unless this snapshot reverts, future imports for this account should be forbidden.
	// MarkStateObjectWritten(addr common.Address, snapId int)

	// NotifyRevertedToSnapshot Notifies the state provider that the data it imported beyond snapId has been reverted.
	NotifyRevertedToSnapshot(snapId int)
}

type ForkedStateError struct {
	CannotQueryDirtyAccount bool
	Error error
}

type ForkedStorageError struct {
	CannotQueryDirtySlot bool
	Error error
}


func (s *ProxyStateDB) Exist(addr common.Address) bool {
	if s.getStateObject(addr) != nil {
		return true
	}
	// This is a bit of a hack because we don't have a way to check this directly through RPC.
	// We're going to use the same definition of empty() as the definition of exist(), and here's why.
	//
	// When querying the various state trie fields through rpc (nonce via GetTransactionCount, balance via GetBalance, Code, etc.),
	// these functions will return "zero" if the state object does not exist OR if the state object field is zero.
	// So essentially we have a way to figure out if a state object is empty, but not if it exists.
	//
	// If you look at the definition of empty, it says that all state objects that don't exist are also empty,
	// but not all empty state objects don't exist. So what happens when there's a state object that's empty and we say it doesn't exist?
	// Does this matter? I'm not sure. I think it'll only impact gas/CREATE behavior.
	return s.Empty(addr)
}

func (s *ProxyStateDB) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	if s.existsLocally(addr) && so.empty() {
		return true
	} else {
		so = s.populateStateObjectFromFork(addr)
		return so.empty()
	}
}

func (s *ProxyStateDB) GetNonce(addr common.Address) uint64 {
	if s.existsLocally(addr) {
		stateObject := s.getStateObject(addr)
		return stateObject.Nonce()
	} else {
		so := s.populateStateObjectFromFork(addr)
		return so.Nonce()
	}
}

func (s *ProxyStateDB) populateStateObjectFromFork(addr common.Address) *stateObject {
	obj := s.getStateObject(addr)
	if obj != nil {
		panic("state objects should be nil when we're trying to populate them")
	}

	bal, nonce, code, err := s.forkStateProvider.ImportStateObject(addr, s.nextRevisionId)
	// NOTE: this function will cause non-existent objects to be created as empty.
	// AFAIK, this only impacts gas.
	// Besides, the journal will delete all empty objects at the end of the transaction during statedb.Finalise, so I
	// suspect it's not a big deal.
	if err != nil {
		if err.CannotQueryDirtyAccount {
			// Something deleted this account, and now we're trying to pull it in from RPC again.
			obj = s.createObject(addr)
			obj.SetBalance(uint256.NewInt(0), tracing.BalanceChangeUnspecified)
			obj.SetNonce(0)
			obj.SetCode(crypto.Keccak256Hash(nil), nil)
			return obj
		} else {
			panic("populate from fork")
		}
	} else {
		obj = s.createObject(addr)
		obj.SetBalance(bal, tracing.BalanceChangeUnspecified)
		obj.SetNonce(nonce)
		obj.SetCode(crypto.Keccak256Hash(code), code)
		return obj
	}
}

func (s *ProxyStateDB) existsLocally(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

func (s *ProxyStateDB) GetBalance(addr common.Address) *uint256.Int {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.Balance()
}

func (s *ProxyStateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.Code()
}

func (s *ProxyStateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.CodeSize()
}

func (s *ProxyStateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (s *ProxyStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		// no stateObject, import it, then populate slot from rpc
		stateObject = s.populateStateObjectFromFork(addr)
	} else {
		data := stateObject.GetState(hash)
		if data != (common.Hash{}) {
			return data
		}
		// if state == 0, we need to check the fork provider to see if we're missing data
	}
	return s.populateSlotFromFork(addr, hash)
}

func (s *ProxyStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		// no stateObject, import it, then populate slot from rpc
		stateObject = s.populateStateObjectFromFork(addr)
	} else {
		data := stateObject.GetCommittedState(hash)
		if data != (common.Hash{}) {
			return data
		}
		// if state == 0, we need to check the fork provider to see if we're missing data
	}
	return s.populateSlotFromFork(addr, hash)
}

func (s *ProxyStateDB) populateSlotFromFork(addr common.Address, hash common.Hash) common.Hash {
	so := s.getStateObject(addr)
	if so == nil {
		panic("populateSlotFromFork called for state object that doesnt exist")
	}

	data, err := s.forkStateProvider.ImportStorageAt(addr, hash, s.nextRevisionId)
	if err != nil {
		if err.CannotQueryDirtySlot {
			// slot was written to by our testing, rpc has old value.
			// data == zero because we wrote the zero
			return common.Hash{}
		} else {
			panic("getState err")
		}
	} else {
		// write & return the fork-provided value
		so.SetState(hash, data)
		return data
	}
}

// HasSelfDestructed Technically, we should be checking the RPC for this. IMO it doesn't really matter because self destruct is basically
// irrelevant for us. Function is only overridden here for this comment.
func (s *ProxyStateDB) HasSelfDestructed(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.selfDestructed
	}
	return false
}

/*
 * SETTERS
 */

func (s *ProxyStateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.AddBalance(amount, reason)
}

func (s *ProxyStateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SubBalance(amount, reason)
}

func (s *ProxyStateDB) SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetBalance(amount, reason)
}

func (s *ProxyStateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetNonce(nonce)
}

func (s *ProxyStateDB) SetCode(addr common.Address, code []byte) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetCode(crypto.Keccak256Hash(code), code)
}

func (s *ProxyStateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}

	stateObject.SetState(key, value)
	s.forkStateProvider.MarkSlotWritten(addr, key, s.nextRevisionId)
}

// RevertToSnapshot reverts all state changes made since the given revision.
// Identical to the built-in, but notifies the state provider of the reversion.
func (s *ProxyStateDB) RevertToSnapshot(revid int) {
	s.StateDB.RevertToSnapshot(revid)
	s.forkStateProvider.NotifyRevertedToSnapshot(revid)
}


