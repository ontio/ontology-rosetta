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
	"fmt"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/ontio/ontology-rosetta/common"
	"github.com/ontio/ontology-rosetta/config"
	"github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/payload"
	bactor "github.com/ontio/ontology/http/base/actor"
)

type MemPoolService struct {
	network *types.NetworkIdentifier
}

// NewNetworkAPIService creates a new instance of a NetworkAPIService.
func NewMemPoolService(network *types.NetworkIdentifier) server.MempoolAPIServicer {
	return &MemPoolService{
		network: network,
	}
}

func (this *MemPoolService) Mempool(ctx context.Context, req *types.NetworkRequest) (*types.MempoolResponse,
	*types.Error) {
	txMap := bactor.GetTxsFromPool(false)
	resp := &types.MempoolResponse{
		TransactionIdentifiers: make([]*types.TransactionIdentifier, 0),
	}
	for hash := range txMap {
		resp.TransactionIdentifiers = append(resp.TransactionIdentifiers, &types.TransactionIdentifier{
			Hash: hash.ToHexString(),
		})
	}
	return resp, nil
}

func (this *MemPoolService) MempoolTransaction(ctx context.Context, req *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	hash, err := common.Uint256FromHexString(req.TransactionIdentifier.Hash)
	if err != nil {
		log.RosettaLog.Errorf("MempoolTransaction: parse req hash %s, %s", req.TransactionIdentifier.Hash, err)
		return nil, TXHASH_INVALID
	}
	tx, err := bactor.GetTxFromPool(hash)
	if err != nil {
		log.RosettaLog.Errorf("MempoolTransaction: %s", err)
		return nil, TX_NOT_EXIST_IN_MEM
	}
	if tx.Tx == nil || tx.Tx.Payload == nil {
		return nil, PARAMS_ERROR
	}
	invokeCode, ok := tx.Tx.Payload.(*payload.InvokeCode)
	if !ok {
		log.RosettaLog.Errorf("MempoolTransaction: invalid tx payload")
		return nil, INVALID_PAYLOAD
	}
	transferState, contract, err := utils.ParsePayload(invokeCode.Code)
	if err != nil {
		log.RosettaLog.Errorf("MempoolTransaction: %s", err)
		return nil, INVALID_PAYLOAD
	}
	currency, ok := utils.Currencies[strings.ToLower(contract.ToHexString())]
	if !ok {
		log.RosettaLog.Errorf("MempoolTransaction: tx currency %s not exist", contract.ToHexString())
		return nil, CURRENCY_NOT_CONFIG
	}
	rosettaTx := &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: req.TransactionIdentifier.Hash},
		Operations:            []*types.Operation{},
	}
	for i, state := range transferState {
		operationFrom := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: 2 * int64(i)},
			Type:                config.OP_TYPE_TRANSFER,
			Status:              config.STATUS_SUCCESS.Status,
			Account: &types.AccountIdentifier{
				Address: state.From.ToBase58(),
			},
			Amount: &types.Amount{
				Value:    fmt.Sprintf("-%d", state.Value),
				Currency: currency,
			},
		}
		operationTo := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: 2*int64(i) + 1},
			RelatedOperations: []*types.OperationIdentifier{
				{Index: operationFrom.OperationIdentifier.Index},
			},
			Type:   config.OP_TYPE_TRANSFER,
			Status: config.STATUS_SUCCESS.Status,
			Account: &types.AccountIdentifier{
				Address: state.To.ToBase58(),
			},
			Amount: &types.Amount{
				Value:    fmt.Sprint(state.Value),
				Currency: currency,
			},
		}
		rosettaTx.Operations = append(rosettaTx.Operations, operationTo, operationFrom)
	}
	return &types.MempoolTransactionResponse{
		Transaction: rosettaTx,
	}, nil
}
