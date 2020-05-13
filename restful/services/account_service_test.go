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
	"testing"

	"github.com/ontio/ontology-rosetta/store"
	util "github.com/ontio/ontology-rosetta/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetHeightFromStore(t *testing.T) {
	db, err := store.NewStore("./acc_store")
	if err != nil {
		t.Error("newStore err:", err)
	}
	balanceInfos := make([]*BalanceInfo, 0)
	balances := make([]*Balance, 0)
	balances = append(balances, &Balance{Height: 7, Amount: 20})
	balances = append(balances, &Balance{Height: 15, Amount: 9})
	balances = append(balances, &Balance{Height: 25, Amount: 15})
	balanceInfos = append(balanceInfos, &BalanceInfo{Key: getAddrKey("AN8JWdUKz5rhpemD61NkAWmS6eb5WXtmq5", util.ONT_ADDRESS), Value: balances})
	var height uint32
	height = 20
	err = batchSaveBalance(db, height, balanceInfos)
	if err != nil {
		t.Error(err)
	}
	h,err := getHeightFromStore(db)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, height, h)
	amount, err := getHistoryBalance(db, 6, "AN8JWdUKz5rhpemD61NkAWmS6eb5WXtmq5", util.ONT_ADDRESS)
	if err != nil {
		t.Error(err)
	}
	t.Logf(":%d", amount)
}
