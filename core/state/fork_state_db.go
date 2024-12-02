package state

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type ForkStateDb struct {
	*StateDB
	remoteStateProvider RemoteStateProvider
}

func NewForkedStateDb(root common.Hash, db Database, remoteStateProvider RemoteStateProvider) (*ForkStateDb, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	sdb := &ForkStateDb{
		remoteStateProvider: remoteStateProvider,
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

func (s *ForkStateDb) Exist(addr common.Address) bool {
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

func (s *ForkStateDb) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	if s.existsLocally(addr) && so.empty() {
		return true
	} else {
		// this might be inadequate if self-destruct is possible

		if s.existsLocally(addr) {
			return false
		} else {
			so = s.populateStateObjectFromFork(addr)
			return so.empty()
		}
	}
}

func (s *ForkStateDb) GetNonce(addr common.Address) uint64 {
	if s.existsLocally(addr) {
		stateObject := s.getStateObject(addr)
		return stateObject.Nonce()
	} else {
		so := s.populateStateObjectFromFork(addr)
		return so.Nonce()
	}
}

func (s *ForkStateDb) populateStateObjectFromFork(addr common.Address) *stateObject {
	obj := s.getStateObject(addr)
	if obj != nil {
		panic("state objects should be nil when we're trying to populate them")
	}

	bal, nonce, code, err := s.remoteStateProvider.ImportStateObject(addr, s.nextRevisionId)
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
			// something is wrong with the RPC, or the fuzzer's context was cancelled.
			// we don't care about propagating the error when the context is cancelled,
			// so do not propagate via dbErr
			if !errors.Is(err.Error, context.Canceled) {
				s.dbErr = err.Error
			}
			return s.createObject(addr)
		}
	} else {
		obj = s.createObject(addr)
		obj.SetBalance(bal, tracing.BalanceChangeUnspecified)
		obj.SetNonce(nonce)
		obj.SetCode(crypto.Keccak256Hash(code), code)
		return obj
	}
}

func (s *ForkStateDb) existsLocally(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

func (s *ForkStateDb) GetBalance(addr common.Address) *uint256.Int {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.Balance()
}

func (s *ForkStateDb) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.Code()
}

func (s *ForkStateDb) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return stateObject.CodeSize()
}

func (s *ForkStateDb) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (s *ForkStateDb) GetState(addr common.Address, hash common.Hash) common.Hash {
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

func (s *ForkStateDb) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
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

func (s *ForkStateDb) populateSlotFromFork(addr common.Address, hash common.Hash) common.Hash {
	so := s.getStateObject(addr)
	if so == nil {
		panic("populateSlotFromFork called for state object that doesnt exist")
	}

	data, err := s.remoteStateProvider.ImportStorageAt(addr, hash, s.nextRevisionId)
	if err != nil {
		if err.CannotQueryDirtySlot {
			// slot was written to by our testing, rpc has old value.
			// data == zero because we wrote the zero
			return common.Hash{}
		} else {
			// something is wrong with the RPC, or the fuzzer's context was cancelled.
			// we don't care about propagating the error when the context is cancelled,
			// so do not propagate via dbErr
			if !errors.Is(err.Error, context.Canceled) {
				s.dbErr = err.Error
			}
			return common.Hash{}
		}
	} else {
		// write & return the fork-provided value
		so.SetState(hash, data)
		return data
	}
}

// HasSelfDestructed Technically, we should be checking the RPC for this. IMO it doesn't really matter because self destruct is basically
// irrelevant for us. Function is only overridden here for this comment.
func (s *ForkStateDb) HasSelfDestructed(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.selfDestructed
	}
	return false
}

/*
 * SETTERS
 */

func (s *ForkStateDb) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.AddBalance(amount, reason)
}

func (s *ForkStateDb) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SubBalance(amount, reason)
}

func (s *ForkStateDb) SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetBalance(amount, reason)
}

func (s *ForkStateDb) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetNonce(nonce)
}

func (s *ForkStateDb) SetCode(addr common.Address, code []byte) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}
	stateObject.SetCode(crypto.Keccak256Hash(code), code)
}

func (s *ForkStateDb) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject = s.populateStateObjectFromFork(addr)
	}

	stateObject.SetState(key, value)
	s.remoteStateProvider.MarkSlotWritten(addr, key, s.nextRevisionId)
}

// RevertToSnapshot reverts all state changes made since the given revision.
// Identical to the built-in, but notifies the state provider of the reversion.
func (s *ForkStateDb) RevertToSnapshot(revid int) {
	s.StateDB.RevertToSnapshot(revid)
	s.remoteStateProvider.NotifyRevertedToSnapshot(revid)
}
