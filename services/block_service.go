/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package services

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/log"
	"github.com/ontio/ontology-rosetta/model"
	"github.com/ontio/ontology/common"
)

// Block implements the /block endpoint.
func (s *service) Block(ctx context.Context, r *types.BlockRequest) (*types.BlockResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	info, xerr := s.store.getBlockInfo(r.BlockIdentifier, true)
	if xerr != nil {
		return nil, xerr
	}
	parent := info.blockID
	if info.height > 0 {
		pinfo, xerr := s.store.getBlockInfoRaw(&blockID{
			byHeight: true,
			height:   info.height - 1,
		}, false)
		if xerr != nil {
			return nil, xerr
		}
		parent = pinfo.blockID
	}
	txs := make([]*types.Transaction, len(info.block.Transactions))
	for i, src := range info.block.Transactions {
		dst, xerr, err := s.transformTransaction(src)
		if xerr != nil {
			return nil, xerr
		}
		if err != nil {
			log.Errorf(
				"Consistency failure when decoding transaction %d at block %d: %s",
				i, info.height, err,
			)
			return nil, wrapErr(errDatastoreConsistency, err)
		}
		txs[i] = dst
	}
	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier:       info.blockID,
			ParentBlockIdentifier: parent,
			Timestamp:             info.blockTimestamp(),
			Transactions:          txs,
		},
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *service) BlockTransaction(ctx context.Context, r *types.BlockTransactionRequest) (*types.BlockTransactionResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	if r.TransactionIdentifier == nil {
		return nil, errInvalidTransactionHash
	}
	txhash, err := common.Uint256FromHexString(r.TransactionIdentifier.Hash)
	if err != nil {
		return nil, errInvalidTransactionHash
	}
	if r.BlockIdentifier == nil {
		return nil, errInvalidBlockIdentifier
	}
	info, xerr := s.store.getBlockInfo(&types.PartialBlockIdentifier{
		Hash:  &r.BlockIdentifier.Hash,
		Index: &r.BlockIdentifier.Index,
	}, true)
	if xerr != nil {
		return nil, xerr
	}
	hash := txhash[:]
	for _, src := range info.block.Transactions {
		if !bytes.Equal(src.Hash, hash) {
			continue
		}
		dst, xerr, err := s.transformTransaction(src)
		if xerr != nil {
			return nil, xerr
		}
		if err != nil {
			log.Errorf(
				"Consistency failure when decoding transaction hash %q at block %d: %s",
				r.TransactionIdentifier.Hash, info.height, err,
			)
			return nil, wrapErr(errDatastoreConsistency, err)
		}
		return &types.BlockTransactionResponse{
			Transaction: dst,
		}, nil
	}
	return nil, errInvalidTransactionHash
}

func (s *service) appendOperations(ops []*types.Operation, xfer *transferInfo) []*types.Operation {
	neg := (&big.Int{}).Neg(xfer.amount)
	related := false
	// NOTE(tav): We specify statusSuccess for all operations, assuming that
	// only the gas fee transfers would have been indexed for transactions
	// that failed.
	if xfer.from != nullAddr {
		op := &types.Operation{
			Account: &types.AccountIdentifier{
				Address: xfer.from.ToBase58(),
			},
			Amount: &types.Amount{
				Currency: xfer.currency,
				Value:    neg.String(),
			},
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Status: &statusSuccess,
			Type:   opTransfer,
		}
		if !xfer.isNative() {
			op.Account.SubAccount = &types.SubAccountIdentifier{
				Address: xfer.contract.ToHexString(),
			}
		}
		if xfer.isGas {
			op.Type = opGasFee
		}
		related = true
		ops = append(ops, op)
	}
	if xfer.to != nullAddr {
		op := &types.Operation{
			Account: &types.AccountIdentifier{
				Address: xfer.to.ToBase58(),
			},
			Amount: &types.Amount{
				Currency: xfer.currency,
				Value:    xfer.amount.String(),
			},
			OperationIdentifier: &types.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Status: &statusSuccess,
			Type:   opTransfer,
		}
		if !xfer.isNative() {
			op.Account.SubAccount = &types.SubAccountIdentifier{
				Address: xfer.contract.ToHexString(),
			}
		}
		if xfer.isGas {
			op.Type = opGasFee
		}
		if related {
			op.RelatedOperations = []*types.OperationIdentifier{
				{Index: int64(len(ops) - 1)},
			}
		}
		ops = append(ops, op)
	}
	return ops
}

func (s *service) transformTransaction(txn *model.Transaction) (*types.Transaction, *types.Error, error) {
	hash, err := common.Uint256ParseFromBytes(txn.Hash)
	if err != nil {
		return nil, nil, fmt.Errorf("services: failed to decode txhash: %s", err)
	}
	ops := []*types.Operation{}
	for _, xfer := range txn.Transfers {
		amount := (&big.Int{}).SetBytes(xfer.Amount)
		contract, err := slice2addr(xfer.Contract)
		if err != nil {
			return nil, nil, fmt.Errorf("services: failed to decode contract address: %s", err)
		}
		info, xerr := s.store.getCurrencyInfo(contract)
		if xerr != nil {
			return nil, xerr, nil
		}
		from, err := slice2addr(xfer.From)
		if err != nil {
			return nil, nil, fmt.Errorf(`services: failed to decode "from" address: %s`, err)
		}
		to, err := slice2addr(xfer.To)
		if err != nil {
			return nil, nil, fmt.Errorf(`services: failed to decode "to" address: %s`, err)
		}
		ops = s.appendOperations(ops, &transferInfo{
			amount:   amount,
			contract: contract,
			currency: info.currency,
			from:     from,
			isGas:    xfer.IsGas,
			to:       to,
		})
	}
	return &types.Transaction{
		Operations: ops,
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash.ToHexString(),
		},
	}, nil, nil
}
