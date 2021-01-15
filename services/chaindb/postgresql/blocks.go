// Copyright © 2020, 2021 Weald Technology Trading.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgresql

import (
	"context"
	"database/sql"

	spec "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	"github.com/wealdtech/chaind/services/chaindb"
)

// SetBlock sets a block.
func (s *Service) SetBlock(ctx context.Context, block *chaindb.Block) error {
	tx := s.tx(ctx)
	if tx == nil {
		return ErrNoTransaction
	}

	var canonical sql.NullBool
	if block.Canonical != nil {
		canonical.Valid = true
		canonical.Bool = *block.Canonical
	}
	_, err := tx.Exec(ctx, `
      INSERT INTO t_blocks(f_slot
                          ,f_proposer_index
                          ,f_root
                          ,f_graffiti
                          ,f_randao_reveal
                          ,f_body_root
                          ,f_parent_root
                          ,f_state_root
                          ,f_canonical
                          ,f_eth1_block_hash
                          ,f_eth1_deposit_count
                          ,f_eth1_deposit_root
						  )
      VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
      ON CONFLICT (f_slot,f_root) DO
      UPDATE
      SET f_proposer_index = excluded.f_proposer_index
         ,f_root = excluded.f_root
         ,f_graffiti = excluded.f_graffiti
         ,f_randao_reveal = excluded.f_randao_reveal
         ,f_state_root = excluded.f_state_root
         ,f_canonical = excluded.f_canonical
         ,f_eth1_block_hash = excluded.f_eth1_block_hash
         ,f_eth1_deposit_count = excluded.f_eth1_deposit_count
         ,f_eth1_deposit_root = excluded.f_eth1_deposit_root
	  `,
		block.Slot,
		block.ProposerIndex,
		block.Root[:],
		block.Graffiti,
		block.RANDAOReveal[:],
		block.BodyRoot[:],
		block.ParentRoot[:],
		block.StateRoot[:],
		canonical,
		block.ETH1BlockHash,
		block.ETH1DepositCount,
		block.ETH1DepositRoot[:],
	)
	if canonical.Valid {
		val := canonical.Bool
		block.Canonical = &val
	}

	return err
}

// BlocksBySlot fetches all blocks with the given slot.
func (s *Service) BlocksBySlot(ctx context.Context, slot spec.Slot) ([]*chaindb.Block, error) {
	tx := s.tx(ctx)
	if tx == nil {
		ctx, cancel, err := s.BeginTx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to begin transaction")
		}
		tx = s.tx(ctx)
		defer cancel()
	}

	rows, err := tx.Query(ctx, `
      SELECT f_slot
            ,f_proposer_index
            ,f_root
            ,f_graffiti
            ,f_randao_reveal
            ,f_body_root
            ,f_parent_root
            ,f_state_root
            ,f_canonical
            ,f_eth1_block_hash
            ,f_eth1_deposit_count
            ,f_eth1_deposit_root
      FROM t_blocks
      WHERE f_slot = $1`,
		slot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]*chaindb.Block, 0)

	for rows.Next() {
		block := &chaindb.Block{}
		var blockRoot []byte
		var randaoReveal []byte
		var bodyRoot []byte
		var parentRoot []byte
		var stateRoot []byte
		var canonical sql.NullBool
		var eth1DepositRoot []byte
		err := rows.Scan(
			&block.Slot,
			&block.ProposerIndex,
			&blockRoot,
			&block.Graffiti,
			&randaoReveal,
			&bodyRoot,
			&parentRoot,
			&stateRoot,
			&canonical,
			&block.ETH1BlockHash,
			&block.ETH1DepositCount,
			&eth1DepositRoot,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		copy(block.Root[:], blockRoot)
		copy(block.RANDAOReveal[:], randaoReveal)
		copy(block.BodyRoot[:], bodyRoot)
		copy(block.ParentRoot[:], parentRoot)
		copy(block.StateRoot[:], stateRoot)
		if canonical.Valid {
			val := canonical.Bool
			block.Canonical = &val
		}
		copy(block.ETH1DepositRoot[:], eth1DepositRoot)
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// BlockByRoot fetches the block with the given root.
func (s *Service) BlockByRoot(ctx context.Context, root spec.Root) (*chaindb.Block, error) {
	tx := s.tx(ctx)
	if tx == nil {
		ctx, cancel, err := s.BeginTx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to begin transaction")
		}
		tx = s.tx(ctx)
		defer cancel()
	}

	block := &chaindb.Block{}
	var blockRoot []byte
	var randaoReveal []byte
	var bodyRoot []byte
	var parentRoot []byte
	var stateRoot []byte
	var canonical sql.NullBool
	var eth1DepositRoot []byte

	err := tx.QueryRow(ctx, `
      SELECT f_slot
            ,f_proposer_index
            ,f_root
            ,f_graffiti
            ,f_randao_reveal
            ,f_body_root
            ,f_parent_root
            ,f_state_root
            ,f_canonical
            ,f_eth1_block_hash
            ,f_eth1_deposit_count
            ,f_eth1_deposit_root
      FROM t_blocks
      WHERE f_root = $1`,
		root[:],
	).Scan(
		&block.Slot,
		&block.ProposerIndex,
		&blockRoot,
		&block.Graffiti,
		&randaoReveal,
		&bodyRoot,
		&parentRoot,
		&stateRoot,
		&canonical,
		&block.ETH1BlockHash,
		&block.ETH1DepositCount,
		&eth1DepositRoot,
	)
	if err != nil {
		return nil, err
	}
	copy(block.Root[:], blockRoot)
	copy(block.RANDAOReveal[:], randaoReveal)
	copy(block.BodyRoot[:], bodyRoot)
	copy(block.ParentRoot[:], parentRoot)
	copy(block.StateRoot[:], stateRoot)
	if canonical.Valid {
		val := canonical.Bool
		block.Canonical = &val
	}
	copy(block.ETH1DepositRoot[:], eth1DepositRoot)
	return block, nil
}

// BlocksByParentRoot fetches the blocks with the given root.
func (s *Service) BlocksByParentRoot(ctx context.Context, parentRoot spec.Root) ([]*chaindb.Block, error) {
	tx := s.tx(ctx)
	if tx == nil {
		ctx, cancel, err := s.BeginTx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to begin transaction")
		}
		tx = s.tx(ctx)
		defer cancel()
	}

	rows, err := tx.Query(ctx, `
      SELECT f_slot
            ,f_proposer_index
            ,f_root
            ,f_graffiti
            ,f_randao_reveal
            ,f_body_root
            ,f_parent_root
            ,f_state_root
            ,f_canonical
            ,f_eth1_block_hash
            ,f_eth1_deposit_count
            ,f_eth1_deposit_root
      FROM t_blocks
      WHERE f_parent_root = $1`,
		parentRoot[:],
	)
	if err != nil {
		return nil, err
	}

	blocks := make([]*chaindb.Block, 0)

	for rows.Next() {
		block := &chaindb.Block{}
		var blockRoot []byte
		var randaoReveal []byte
		var bodyRoot []byte
		var parentRoot []byte
		var stateRoot []byte
		var canonical sql.NullBool
		var eth1DepositRoot []byte
		err := rows.Scan(
			&block.Slot,
			&block.ProposerIndex,
			&blockRoot,
			&block.Graffiti,
			&randaoReveal,
			&bodyRoot,
			&parentRoot,
			&stateRoot,
			&canonical,
			&block.ETH1BlockHash,
			&block.ETH1DepositCount,
			&eth1DepositRoot,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		copy(block.Root[:], blockRoot)
		copy(block.RANDAOReveal[:], randaoReveal)
		copy(block.BodyRoot[:], bodyRoot)
		copy(block.ParentRoot[:], parentRoot)
		copy(block.StateRoot[:], stateRoot)
		if canonical.Valid {
			val := canonical.Bool
			block.Canonical = &val
		}
		copy(block.ETH1DepositRoot[:], eth1DepositRoot)
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// EmptySlots fetches the slots in the given range without a block in the database.
func (s *Service) EmptySlots(ctx context.Context, minSlot spec.Slot, maxSlot spec.Slot) ([]spec.Slot, error) {
	tx := s.tx(ctx)
	if tx == nil {
		ctx, cancel, err := s.BeginTx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to begin transaction")
		}
		tx = s.tx(ctx)
		defer cancel()
	}

	rows, err := tx.Query(ctx, `
      SELECT missed
	  FROM generate_series($1,$2,1) missed
	  LEFT JOIN t_blocks
	  ON missed = t_blocks.f_slot
	  WHERE f_slot IS NULL`,
		minSlot,
		maxSlot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	missedSlots := make([]spec.Slot, 0)
	for rows.Next() {
		missedSlot := spec.Slot(0)
		err := rows.Scan(&missedSlot)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		missedSlots = append(missedSlots, missedSlot)
	}

	return missedSlots, nil
}

// IndeterminateBlocks fetches the blocks in the given range that do not have a canonical status.
func (s *Service) IndeterminateBlocks(ctx context.Context, minSlot spec.Slot, maxSlot spec.Slot) ([]spec.Root, error) {
	tx := s.tx(ctx)
	if tx == nil {
		ctx, cancel, err := s.BeginTx(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to begin transaction")
		}
		tx = s.tx(ctx)
		defer cancel()
	}

	rows, err := tx.Query(ctx, `
      SELECT f_root
      FROM t_blocks
      WHERE f_slot >= $1
        AND f_slot < $2
        AND f_canonical IS NULL
	  ORDER BY f_slot`,
		minSlot,
		maxSlot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indeterminateRoots := make([]spec.Root, 0)
	var missedRootBytes []byte
	for rows.Next() {
		err := rows.Scan(&missedRootBytes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		var missedRoot spec.Root
		copy(missedRoot[:], missedRootBytes)
		indeterminateRoots = append(indeterminateRoots, missedRoot)
	}

	return indeterminateRoots, nil
}
