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
	"time"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/version"
	"github.com/ontio/ontology/p2pserver/common"
)

// NetworkList implements the /network/list endpoint.
func (s *service) NetworkList(ctx context.Context, r *types.MetadataRequest) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: s.networks,
	}, nil
}

// NetworkOptions implements the /network/options endpoint.
func (s *service) NetworkOptions(ctx context.Context, r *types.NetworkRequest) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Allow: &types.Allow{
			Errors:                  serverErrors,
			HistoricalBalanceLookup: true,
			OperationStatuses: []*types.OperationStatus{
				{Status: statusSuccess, Successful: true},
				{Status: statusFailed, Successful: false},
			},
			OperationTypes: opTypes,
		},
		Version: &types.Version{
			NodeVersion:    version.Node,
			RosettaVersion: version.Rosetta,
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint.
func (s *service) NetworkStatus(ctx context.Context, r *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	height := s.store.getHeight()
	cur, xerr := s.store.getBlockInfoRaw(&blockID{
		byHeight: true,
		height:   height,
	}, true)
	if xerr != nil {
		return nil, xerr
	}
	genesis, xerr := s.store.getBlockInfoRaw(&blockID{
		byHeight: true,
		height:   0,
	}, false)
	if xerr != nil {
		return nil, xerr
	}
	peers := []*types.Peer{}
	network := s.node.GetNetwork()
	self := peerid2hex(network.GetID())
	for _, peer := range network.GetNeighbors() {
		metadata := map[string]interface{}{
			"address":      peer.GetAddr(),
			"height":       peer.GetHeight(),
			"last_contact": peer.GetContactTime().UTC().Format(time.RFC3339),
			"relay":        peer.GetRelay(),
			"self":         self,
			"version":      peer.GetSoftVersion(),
		}
		peers = append(peers, &types.Peer{
			PeerID:   peerid2hex(peer.GetID()),
			Metadata: metadata,
		})
	}
	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: cur.blockID,
		CurrentBlockTimestamp:  cur.blockTimestamp(),
		GenesisBlockIdentifier: genesis.blockID,
		Peers:                  peers,
		SyncStatus:             s.store.syncStatus(),
	}, nil
}

func peerid2hex(id common.PeerId) string {
	return id.ToHexString()
}
