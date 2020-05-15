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
	"fmt"
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/common"
	"github.com/ontio/ontology-rosetta/restful/services"
	db "github.com/ontio/ontology-rosetta/store"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/p2pserver"
)

// NewBlockchainRouter creates a Mux http.Handler from a collection
// of server controllers.
var (
	BLOCKCHAIN = "ont"
	MAINNET    = "mainnet"
	TESTNET    = "testnet"
	PRIVATENET = "privatenet"
)

func NewBlockchainRouter(
	network *types.NetworkIdentifier,
	asserter *asserter.Asserter,
	p2pSvr *p2pserver.P2PServer,
	store *db.Store,
) http.Handler {
	accountAPIService := services.NewAccountAPIService(network, store)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
	)

	blockAPIService := services.NewBlockAPIService(network)
	blockAPIController := server.NewBlockAPIController(blockAPIService, asserter)

	constructAPIService := services.NewConstructionAPIService(network, store)
	constructAPIController := server.NewConstructionAPIController(constructAPIService, asserter)

	networkAPIService := services.NewNetworkAPIService(network, p2pSvr)
	networtAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	mempoolAPIService := services.NewMemPoolService(network)
	mempoolAPIController := server.NewMempoolAPIController(mempoolAPIService, asserter)

	return server.NewRouter(accountAPIController, blockAPIController, constructAPIController, networtAPIController,
		mempoolAPIController)
}

func NewService(restfulPort int32, p2pSvr *p2pserver.P2PServer, store *db.Store) error {
	networkName := "unnkown"
	if config.DefConfig.P2PNode.NetworkName == config.NETWORK_NAME_MAIN_NET {
		networkName = MAINNET
	} else if config.DefConfig.P2PNode.NetworkName == config.NETWORK_NAME_POLARIS_NET {
		networkName = TESTNET
	} else if config.DefConfig.P2PNode.NetworkName == config.NETWORK_NAME_SOLO_NET {
		networkName = PRIVATENET
	}
	network := &types.NetworkIdentifier{
		Blockchain: BLOCKCHAIN,
		Network:    networkName,
	}
	// The asserter automatically rejects incorrectly formatted
	// requests.
	asserter, err := asserter.NewServer([]*types.NetworkIdentifier{network})
	if err != nil {
		common.RosetaaLog.Fatal(err)
		return err
	}
	router := NewBlockchainRouter(network, asserter, p2pSvr, store)
	return http.ListenAndServe(fmt.Sprintf(":%d", restfulPort), router)
}
