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

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology/common"
)

// AccountBalance implements the /account/balance endpoint.
func (s *service) AccountBalance(ctx context.Context, r *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	if r.AccountIdentifier == nil {
		return nil, errInvalidAccountAddress
	}
	acct, err := common.AddressFromBase58(r.AccountIdentifier.Address)
	if err != nil {
		return nil, errInvalidAccountAddress
	}
	if r.AccountIdentifier.SubAccount == nil {
		return s.store.getBalance(r.BlockIdentifier, acct, r.Currencies, ontAddr, ongAddr)
	}
	contract, err := common.AddressFromHexString(r.AccountIdentifier.SubAccount.Address)
	if err != nil {
		return nil, errInvalidContractAddress
	}
	return s.store.getBalance(r.BlockIdentifier, acct, r.Currencies, contract)
}

// AccountCoins implements the /account/coins endpoint.
func (s *service) AccountCoins(ctx context.Context, r *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	return nil, errNotImplemented
}
