package stagedsync

import (
	"context"
	"fmt"

	"github.com/harmony-one/harmony/core"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/ledgerwatch/erigon-lib/kv"
)

// The number of blocks we should be able to re-org sub-second on commodity hardware.
// See https://hackmd.io/TdJtNs0dS56q-In8h-ShSg
const ShortPoSReorgThresholdBlocks = 10

type StageHeaders struct {
	configs StageHeadersCfg
}

type StageHeadersCfg struct {
	ctx context.Context
	bc  *core.BlockChain
	db  kv.RwDB
}

func NewStageHeders(cfg StageHeadersCfg) *StageHeaders {
	return &StageHeaders{
		configs: cfg,
	}
}

func NewStageHeadersCfg(ctx context.Context, bc *core.BlockChain, db kv.RwDB) StageHeadersCfg {
	return StageHeadersCfg{
		ctx: ctx,
		bc:  bc,
		db:  db,
	}
}

func (headers *StageHeaders) Exec(firstCycle bool, badBlockUnwind bool, s *StageState, unwinder Unwinder, tx kv.RwTx) error {

	if len(s.state.syncConfig.peers) == 0 {
		return fmt.Errorf("haven't connected to any peer yet!")
	}
	otherHeight := uint64(0)
	if errV := headers.configs.db.View(headers.configs.ctx, func(etx kv.Tx) (err error) {
		if otherHeight, err = s.CurrentStageProgress(etx); err != nil {
			return err
		}
		return nil
	}); errV != nil {
		return errV
	}

	useExternalTx := tx != nil
	if !useExternalTx {
		var err error
		tx, err = headers.configs.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}
	fmt.Println("current block height:", headers.configs.bc.CurrentBlock().NumberU64())
	if otherHeight <= headers.configs.bc.CurrentBlock().NumberU64() {
		maxPeersHeight, err := s.state.getMaxPeerHeight(s.state.IsBeacon())
		if err != nil {
			return err
		}
		fmt.Println("max peers height:", maxPeersHeight)
		if maxPeersHeight <= headers.configs.bc.CurrentBlock().NumberU64() {
			s.state.Done()
			return nil
		}
		otherHeight = maxPeersHeight
	}
	s.state.syncStatus.maxPeersHeight = otherHeight
	
	currentHeight := headers.configs.bc.CurrentBlock().NumberU64()
	if currentHeight >= otherHeight {
		utils.Logger().Info().
			Msgf("[STAGED_SYNC] Node is now IN SYNC! (isBeacon: %t, ShardID: %d, otherHeight: %d, currentHeight: %d)",
				s.state.IsBeacon(), headers.configs.bc.ShardID(), otherHeight, currentHeight)
		s.state.Done()
		return nil
	}

	// GetStageProgress(headers.configs.db, Headers)
	maxBlocksPerSyncCycle := s.state.MaxBlocksPerSyncCycle
	fmt.Println("max blocks per sync cycle is", maxBlocksPerSyncCycle)
	targetHeight := otherHeight
	if maxBlocksPerSyncCycle > 0 && targetHeight-currentHeight > maxBlocksPerSyncCycle {
		targetHeight = currentHeight + maxBlocksPerSyncCycle
	}
	if err := s.Update(tx, targetHeight); err != nil { //SaveStageProgress(tx, Headers, s.state.isBeacon, targetHeight); err != nil {
		utils.Logger().Info().
			Msgf("[STAGED_SYNC] saving progress for headers stage failed: %v", err)
		return err
	}

	if !useExternalTx {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (headers *StageHeaders) Unwind(firstCycle bool, u *UnwindState, s *StageState, tx kv.RwTx) (err error) {
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = headers.configs.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}
	// TODO: Delete canonical hashes that are being unwound

	if !useExternalTx {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (headers *StageHeaders) Prune(firstCycle bool, p *PruneState, tx kv.RwTx) (err error) {
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = headers.configs.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
