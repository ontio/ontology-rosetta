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
	"context"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/ontio/ontology-rosetta/common"
	"github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/common"
	ctypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/http/base/actor"
)

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	network *types.NetworkIdentifier
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(network *types.NetworkIdentifier) server.BlockAPIServicer {
	return &BlockAPIService{
		network: network,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if request.BlockIdentifier.Index == nil && request.BlockIdentifier.Hash == nil {
		return nil, BLOCK_IDENTIFIER_NIL
	}

	if request.BlockIdentifier.Index != nil && *request.BlockIdentifier.Index < 0 {
		return nil, BLOCK_NUMBER_INVALID
	}

	var block *ctypes.Block
	var err error
	if request.BlockIdentifier.Index != nil {
		block, err = actor.GetBlockByHeight(uint32(*request.BlockIdentifier.Index))
		if err != nil {
			log.RosettaLog.Errorf("[Block]GetBlockByHeight failed: %s", err.Error())
			return nil, GET_BLOCK_FAILED
		}
	} else {
		hash, err := common.Uint256FromHexString(*request.BlockIdentifier.Hash)
		if err != nil {
			log.RosettaLog.Errorf("[Block]Uint256FromHexString failed: %s", err.Error())
			return nil, GET_BLOCK_FAILED
		}
		block, err = actor.GetBlockFromStore(hash)
		if err != nil {
			log.RosettaLog.Errorf("[Block]GetBlockFromStore failed: %s", err.Error())
			return nil, GET_BLOCK_FAILED
		}
	}

	if block == nil {
		return nil, UNKNOWN_BLOCK
	}

	//validate block hash
	blocknum := block.Header.Height
	tmphash := block.Hash()
	if request.BlockIdentifier.Hash != nil && *request.BlockIdentifier.Hash != tmphash.ToHexString() {
		return nil, BLOCK_HASH_INVALID
	}
	blockIdentifier := &types.BlockIdentifier{
		Index: int64(block.Header.Height),
		Hash:  tmphash.ToHexString(),
	}

	//ignore genesis block tx
	//if *request.BlockIdentifier.Index == 0 {
	//	return &types.BlockResponse{
	//		Block: &types.Block{
	//			BlockIdentifier:       blockIdentifier,
	//			ParentBlockIdentifier: blockIdentifier,
	//			Timestamp:             int64(block.Header.Timestamp) * 1000,
	//			Transactions:          nil,
	//			//Metadata:              nil,
	//		},
	//		OtherTransactions: nil,
	//	}, nil
	//}

	var parentblock *types.BlockIdentifier
	if blocknum == 0 {
		//genesis block,no parent
		parentblock = blockIdentifier
	} else {
		parentblock = &types.BlockIdentifier{}

		privblock, err := actor.GetBlockByHeight(blocknum - 1)
		if err != nil {
			log.RosettaLog.Errorf("[Block]GetBlockByHeight failed: %s", err.Error())
			return nil, GET_BLOCK_FAILED
		}
		if privblock != nil {
			parentblock.Index = int64(blocknum - 1)
			tmphash := privblock.Hash()
			parentblock.Hash = tmphash.ToHexString()
		}
	}

	txs := make([]*types.Transaction, 0)
	othertxs := make([]*types.TransactionIdentifier, 0)
	for _, tx := range block.Transactions {
		rtx, err := utils.TransformTransaction(tx)
		if err != nil {
			log.RosettaLog.Errorf("[Block]TransformTransaction failed: %s", err.Error())
			return nil, GET_TRANSACTION_FAILED
		}
		if len(rtx.Operations) > 0 {
			txs = append(txs, rtx)
		} else {
			txhash := tx.Hash()
			othertxs = append(othertxs, &types.TransactionIdentifier{Hash: txhash.ToHexString()})
		}

	}

	rblock := &types.Block{
		BlockIdentifier:       blockIdentifier,
		ParentBlockIdentifier: parentblock,
		Timestamp:             int64(block.Header.Timestamp) * 1000,
		Transactions:          txs,
		//Metadata:              nil,
	}

	return &types.BlockResponse{
		Block:             rblock,
		OtherTransactions: othertxs,
	}, nil

}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	blocknum := request.BlockIdentifier.Index
	blockhash := request.BlockIdentifier.Hash

	txhash, err := common.Uint256FromHexString(request.TransactionIdentifier.Hash)
	if err != nil {
		log.RosettaLog.Errorf("[BlockTransaction]Uint256FromHexString failed: %s", err.Error())
		return nil, TXHASH_INVALID
	}

	blockheight, tx, err := actor.GetTxnWithHeightByTxHash(txhash)
	if err != nil {
		log.RosettaLog.Errorf("[BlockTransaction]GetTxnWithHeightByTxHash failed: %s", err.Error())
		return nil, GET_TRANSACTION_FAILED
	}
	if blocknum != int64(blockheight) {
		return nil, BLOCK_NUMBER_INVALID
	}

	bhash := actor.GetBlockHashFromStore(blockheight)
	if bhash.ToHexString() != blockhash {
		return nil, BLOCK_HASH_INVALID
	}

	rtx, err := utils.TransformTransaction(tx)
	if err != nil {
		log.RosettaLog.Errorf("[BlockTransaction]TransformTransaction failed: %s", err.Error())
		return nil, GET_TRANSACTION_FAILED
	}

	return &types.BlockTransactionResponse{
		Transaction: rtx,
	}, nil
}
