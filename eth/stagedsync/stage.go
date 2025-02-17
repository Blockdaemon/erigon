package stagedsync

import (
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/ethdb"
)

// ExecFunc is the execution function for the stage to move forward.
// * state - is the current state of the stage and contains stage data.
// * unwinder - if the stage needs to cause unwinding, `unwinder` methods can be used.
type ExecFunc func(firstCycle bool, s *StageState, unwinder Unwinder, tx ethdb.RwTx) error

// UnwindFunc is the unwinding logic of the stage.
// * unwindState - contains information about the unwind itself.
// * stageState - represents the state of this stage at the beginning of unwind.
type UnwindFunc func(firstCycle bool, u *UnwindState, s *StageState, tx ethdb.RwTx) error

// PruneFunc is the execution function for the stage to prune old data.
// * state - is the current state of the stage and contains stage data.
type PruneFunc func(firstCycle bool, p *PruneState, tx ethdb.RwTx) error

// Stage is a single sync stage in staged sync.
type Stage struct {
	// Description is a string that is shown in the logs.
	Description string
	// DisabledDescription shows in the log with a message if the stage is disabled. Here, you can show which command line flags should be provided to enable the page.
	DisabledDescription string
	// Forward is called when the stage is executed. The main logic of the stage should be here. Should always end with `s.Done()` to allow going to the next stage. MUST NOT be nil!
	Forward ExecFunc
	// Unwind is called when the stage should be unwound. The unwind logic should be there. MUST NOT be nil!
	Unwind UnwindFunc
	Prune  PruneFunc
	// ID of the sync stage. Should not be empty and should be unique. It is recommended to prefix it with reverse domain to avoid clashes (`com.example.my-stage`).
	ID stages.SyncStage
	// Disabled defines if the stage is disabled. It sets up when the stage is build by its `StageBuilder`.
	Disabled bool
}

// StageState is the state of the stage.
type StageState struct {
	state       *Sync
	ID          stages.SyncStage
	BlockNumber uint64 // BlockNumber is the current block number of the stage at the beginning of the state execution.
}

func (s *StageState) LogPrefix() string { return s.state.LogPrefix() }

// Update updates the stage state (current block number) in the database. Can be called multiple times during stage execution.
func (s *StageState) Update(db ethdb.Putter, newBlockNum uint64) error {
	return stages.SaveStageProgress(db, s.ID, newBlockNum)
}

// ExecutionAt gets the current state of the "Execution" stage, which block is currently executed.
func (s *StageState) ExecutionAt(db ethdb.KVGetter) (uint64, error) {
	execution, err := stages.GetStageProgress(db, stages.Execution)
	return execution, err
}

// Unwinder allows the stage to cause an unwind.
type Unwinder interface {
	// UnwindTo begins staged sync unwind to the specified block.
	UnwindTo(unwindPoint uint64, badBlock common.Hash)
}

// UnwindState contains the information about unwind.
type UnwindState struct {
	ID stages.SyncStage
	// UnwindPoint is the block to unwind to.
	UnwindPoint        uint64
	CurrentBlockNumber uint64
	// If unwind is caused by a bad block, this hash is not empty
	BadBlock common.Hash
	state    *Sync
}

func (u *UnwindState) LogPrefix() string { return u.state.LogPrefix() }

// Done updates the DB state of the stage.
func (u *UnwindState) Done(db ethdb.Putter) error {
	return stages.SaveStageProgress(db, u.ID, u.UnwindPoint)
}

type PruneState struct {
	ID                 stages.SyncStage
	PrunePoint         uint64 // PrunePoint is the block to prune to.
	CurrentBlockNumber uint64
	state              *Sync
}

func (u *PruneState) LogPrefix() string { return u.state.LogPrefix() }
