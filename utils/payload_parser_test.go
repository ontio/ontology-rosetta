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
package utils

import (
	"encoding/hex"
	"testing"

	"github.com/ontio/ontology/v2/common"
	"github.com/ontio/ontology/v2/core/utils"
)

func TestParsePayload(t *testing.T) {
	// https://explorer.ont.io/transaction/e845be647abb86efed9f68e2291e537d77a776f302876f7fa8d3ab860a0b4f30
	oep4TransferPayload, _ := hex.DecodeString("04003cef1514b80aeab7df922939c67eb610731a0235519027be14666d55e5ff" +
		"abc31e3aa72469a0f5bd8c276b5dc353c1087472616e73666572678ae65a5bc55defe3eaf1dc9f68623074e3587bc2")
	states, contract, err := ParsePayload(oep4TransferPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("oep4 transfer: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
	contractAddr, _ := common.AddressFromBase58("AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK")
	spender, _ := common.AddressFromBase58("AVpuXX3mZbjbqJ16weWzbkABxuTRuGiXbf")
	from, _ := common.AddressFromBase58("ASUpHyd8hsTMxKT7pCdPf1dYCZUvov2rk5")
	to, _ := common.AddressFromBase58("AYZ14K5FJKXC9mzS5YFfdr52E6seBqAPPU")
	amount := uint64(18289182)
	oep4TransferFromPayload, err := utils.BuildNeoVMInvokeCode(contractAddr, []interface{}{"transferFrom",
		[]interface{}{spender, from, to, amount}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("oep4 transferFrom payload: %x", oep4TransferFromPayload)
	states, contract, err = ParsePayload(oep4TransferFromPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("oep4 transferFrom: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
	type multiTransferState struct {
		From  common.Address
		To    common.Address
		Value uint64
	}
	multiTransferParam := []*multiTransferState{
		{From: from, To: to, Value: amount},
		{From: from, To: to, Value: amount + 2},
		{From: from, To: to, Value: amount + 3},
	}
	oep4TransferMultiPayload, err := utils.BuildNeoVMInvokeCode(contractAddr, []interface{}{"transferMulti",
		[]interface{}{multiTransferParam}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("oep4 transferMulti payload: %x", oep4TransferMultiPayload)
	states, contract, err = ParsePayload(oep4TransferMultiPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("oep4 transfer multi: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
	// https://explorer.ont.io/transaction/2c5d95e532aad1c2d59d6544e5828202a56a61f63c9e2fd098c6c26f86b20d66
	ontTransferPayload, _ := hex.DecodeString("00c66b1473e1e106a810f63501c4399dd58cba2f363eabba6a7cc8145f32857a94" +
		"eaf5eccbf47fd5b9824fb87ecb80fc6a7cc801416a7cc86c51c1087472616e736665721400000000000000000000000000000000000" +
		"000010068164f6e746f6c6f67792e4e61746976652e496e766f6b65")
	states, contract, err = ParsePayload(ontTransferPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("ont transfer: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
	ontTransferPayload, _ = utils.BuildNativeInvokeCode(contractAddr, 00, "transfer",
		[]interface{}{multiTransferParam})
	states, contract, err = ParsePayload(ontTransferPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("ont trasnfer multi: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
	transferFrom := &TransferState{
		Spender: spender,
		To:      to,
		From:    from,
		Value:   amount,
	}
	ontTransferFromPayload, err := utils.BuildNativeInvokeCode(contractAddr, 00, "transferFrom",
		[]interface{}{transferFrom})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ont transferFrom payload: %x", ontTransferFromPayload)
	states, contract, err = ParsePayload(ontTransferFromPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		t.Logf("ont transferFrom: contract %s, from %s, to %s, amount %d",
			contract.ToHexString(), state.From.ToBase58(), state.To.ToBase58(), state.Value)
	}
}
