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

// Package services implements the APIs for the Rosetta Server.
package services

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/model"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/p2pserver"
	"github.com/ontio/ontology/smartcontract/service/neovm"
)

const (
	defaultGasPrice = 2500
	opBurn          = "burn"
	opGasFee        = "gas_fee"
	opMint          = "mint"
	opTransfer      = "transfer"
)

var (
	// The genesis block mints 10^9 ONT (decimals 0) and 10^9 ONG (decimals 9).
	govAddr  = mustHexAddr("0700000000000000000000000000000000000000") // AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK
	nullAddr = mustHexAddr("0000000000000000000000000000000000000000") // AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM
	ongAddr  = mustHexAddr("0200000000000000000000000000000000000000") // AFmseVrdL9f9oyCzZefL9tG6UbvhfRZMHJ
	ontAddr  = mustHexAddr("0100000000000000000000000000000000000000") // AFmseVrdL9f9oyCzZefL9tG6UbvhUMqNMV
)

var (
	minGasLimit   = neovm.MIN_TRANSACTION_GAS
	opTypes       = []string{opBurn, opGasFee, opMint, opTransfer}
	statusFailed  = "FAILED"
	statusSuccess = "SUCCESS"
)

// OEP4Token defines the currency information for an OEP4 token.
type OEP4Token struct {
	Contract common.Address
	Decimals int32
	Symbol   string
	Wasm     bool
}

// IndexConfig represents the options for the IndexBlocks method on Store.
type IndexConfig struct {
	Done      chan bool
	ExitEarly bool
	WaitTime  time.Duration
}

type balanceChange struct {
	diff   *big.Int
	key    []byte
	prefix []byte
}

type blockID struct {
	byHeight bool
	hash     common.Uint256
	height   uint32
}

func (b *blockID) String() string {
	return fmt.Sprintf(
		"block<%v, %d, %s>",
		b.byHeight, b.height, b.hash.ToHexString(),
	)
}

type blockInfo struct {
	block   *model.Block
	blockID *types.BlockIdentifier
	height  uint32
	hval    []byte
}

func (b *blockInfo) blockTimestamp() int64 {
	return int64(b.block.Timestamp) * 1000
}

type blockState struct {
	block   *model.Block
	changes []*balanceChange
	hashes  [][]byte
	id      *blockID
	synced  uint32
}

type currencyInfo struct {
	contract common.Address
	currency *types.Currency
	wasm     bool
}

func (c *currencyInfo) isNative() bool {
	return c.contract == ongAddr || c.contract == ontAddr
}

type service struct {
	networks []*types.NetworkIdentifier
	node     *p2pserver.P2PServer
	offline  bool
	store    *Store
}

type transfer struct {
	amount *big.Int
	from   common.Address
	isGas  bool
	to     common.Address
}

type transferInfo struct {
	amount   *big.Int
	contract common.Address
	currency *types.Currency
	from     common.Address
	isGas    bool
	to       common.Address
}

func (t *transferInfo) isNative() bool {
	return t.contract == ongAddr || t.contract == ontAddr
}

// Router creates an http.Handler for Rosetta API requests.
func Router(node *p2pserver.P2PServer, store *Store, offline bool) (http.Handler, error) {
	networks := []*types.NetworkIdentifier{{
		Blockchain: "ontology",
		Network:    networkName(),
	}}
	asserter, err := asserter.NewServer(
		opTypes,
		true,
		networks,
		nil,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"services: failed to instantiate the Rosetta asserter: %w",
			err,
		)
	}
	svc := &service{
		networks: networks,
		node:     node,
		offline:  offline,
		store:    store,
	}
	return server.NewRouter(
		server.NewAccountAPIController(svc, asserter),
		server.NewBlockAPIController(svc, asserter),
		server.NewConstructionAPIController(svc, asserter),
		server.NewMempoolAPIController(svc, asserter),
		server.NewNetworkAPIController(svc, asserter),
	), nil
}

func mustHexAddr(s string) common.Address {
	addr, err := common.AddressFromHexString(s)
	if err != nil {
		panic(fmt.Errorf("services: invalid hex address %q: %s", s, err))
	}
	return addr
}

func networkName() string {
	switch config.DefConfig.P2PNode.NetworkName {
	case config.NETWORK_NAME_MAIN_NET:
		return "mainnet"
	case config.NETWORK_NAME_POLARIS_NET:
		return "testnet"
	case config.NETWORK_NAME_SOLO_NET:
		return "privatenet"
	default:
		return "unknown"
	}
}
