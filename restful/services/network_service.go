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

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/config"
	"github.com/ontio/ontology/http/base/actor"
	"github.com/ontio/ontology/p2pserver"
)

type NetworkAPIService struct {
	network *types.NetworkIdentifier
	p2pSvr  *p2pserver.P2PServer
}

//Get List of Available Networks. endpoint: /network/list
func (n NetworkAPIService) NetworkList(ctx context.Context, request *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {

	netidentifiers := []*types.NetworkIdentifier{n.network}

	return &types.NetworkListResponse{
		NetworkIdentifiers: netidentifiers,
	}, nil
}

//Get Network Options. endpoint: /network/options
func (n NetworkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	v := &types.Version{
		RosettaVersion:    config.Conf.Rosetta.Version,
		NodeVersion:       config.ONTOLOGY_VERSION,
		MiddlewareVersion: nil,
		Metadata:          nil,
	}

	optstatus := []*types.OperationStatus{
		config.STATUS_SUCCESS,
		config.STATUS_FAILED,
	}

	operationTypes := []string{config.OP_TYPE_TRANSFER}

	errors := []*types.Error{
		NETWORK_IDENTIFIER_ERROR,
		BLOCK_IDENTIFIER_NIL,
		BLOCK_NUMBER_INVALID,
		GET_BLOCK_FAILED,
		BLOCK_HASH_INVALID,
		GET_TRANSACTION_FAILED,
		SIGNED_TX_INVALID,
		COMMIT_TX_FAILED,
		TXHASH_INVALID,
		UNKNOWN_BLOCK,
		SERVER_NOT_SUPPORT,
		ADDRESS_INVALID,
		BALANCE_ERROR,
		PARSE_INT_ERROR,
		JSON_MARSHAL_ERROR,
		INVALID_PAYLOAD,
		CURRENCY_NOT_CONFIG,
		PARAMS_ERROR,
		CONTRACT_ADDRESS_ERROR,
		PRE_EXECUTE_ERROR,
		QUERY_BALANCE_ERROR,
	}

	allow := &types.Allow{
		OperationStatuses: optstatus,
		OperationTypes:    operationTypes,
		Errors:            errors,
	}

	return &types.NetworkOptionsResponse{
		Version: v,
		Allow:   allow,
	}, nil
}

//Get Network Status. endpoint: /network/status
func (n NetworkAPIService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	currentheight := actor.GetCurrentBlockHeight()
	currentblock, err := actor.GetBlockByHeight(currentheight)
	if err != nil {
		return nil, GET_BLOCK_FAILED
	}
	blockhash := currentblock.Hash()
	cbi := &types.BlockIdentifier{
		Index: int64(currentheight),
		Hash:  blockhash.ToHexString(),
	}
	currentBlockTimestamp := int64(currentblock.Header.Timestamp) * 1000
	genesisBlock, err := actor.GetBlockByHeight(0)
	if err != nil {
		return nil, GET_BLOCK_FAILED
	}
	gbHash := genesisBlock.Hash()
	gbi := &types.BlockIdentifier{
		Index: 0,
		Hash:  gbHash.ToHexString(),
	}

	peers := make([]*types.Peer, 0)
	if n.p2pSvr != nil {
		for _, peer := range n.p2pSvr.GetNetWork().GetNeighbors() {
			metadata := make(map[string]interface{})
			metadata["address"] = peer.GetAddr()
			metadata["height"] = peer.GetHeight()
			metadata["state"] = peer.GetState()
			p := &types.Peer{
				PeerID:   fmt.Sprintf("%d", peer.GetID()),
				Metadata: metadata,
			}
			peers = append(peers, p)
		}
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: cbi,
		CurrentBlockTimestamp:  currentBlockTimestamp,
		GenesisBlockIdentifier: gbi,
		Peers:                  peers,
	}, nil
}

func NewNetworkAPIService(network *types.NetworkIdentifier, p2pSvr *p2pserver.P2PServer) server.NetworkAPIServicer {
	return &NetworkAPIService{
		network: network,
		p2pSvr:  p2pSvr,
	}
}
