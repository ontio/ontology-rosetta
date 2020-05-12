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
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	ctypes "github.com/ontio/ontology/core/types"
	ontErrors "github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/http/base/actor"
	bcomn "github.com/ontio/ontology/http/base/common"
)

type ConstructionAPIService struct {
	network *types.NetworkIdentifier
}

func NewConstructionAPIService(network *types.NetworkIdentifier) server.ConstructionAPIServicer {
	return &ConstructionAPIService{network: network}
}

//Get Transaction Construction Metadata. endpoint:/construction/metadata
func (c ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	//todo define the options
	//ni := request.NetworkIdentifier
	//opt := request.Options

	//return the current block height and hash
	//following by the example
	height := actor.GetCurrentBlockHeight()
	hash := actor.CurrentBlockHash()

	metadata := make(map[string]interface{})
	metadata["current_block_hash"] = hash.ToHexString()
	metadata["current_block_height"] = height

	resp := &types.ConstructionMetadataResponse{Metadata: metadata}

	return resp, nil
}

//Submit a Signed Transaction .endpoint: /construction/submit
func (c ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.ConstructionSubmitResponse, *types.Error) {
	//ni := request.NetworkIdentifier
	txStr := request.SignedTransaction
	if len(txStr) == 0 {
		return nil, SIGNED_TX_INVALID
	}

	txbytes, err := common.HexToBytes(txStr)
	if err != nil {
		return nil, SIGNED_TX_INVALID
	}
	txn, err := ctypes.TransactionFromRawBytes(txbytes)
	if err != nil {
		return nil, SIGNED_TX_INVALID
	}
	if errCode, desc := bcomn.SendTxToPool(txn); errCode != ontErrors.ErrNoError {
		log.Debug("[ConstructionSubmit]SendTxToPool failed:%s", desc)
		return nil, COMMIT_TX_FAILED
	}

	txhash := txn.Hash()

	return &types.ConstructionSubmitResponse{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: txhash.ToHexString()},
		Metadata:              nil,
	}, nil
}
