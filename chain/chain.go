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

// Package chain provides functions for dealing with Ontology smart contracts.
package chain

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/ledger"
	hcommon "github.com/ontio/ontology/http/base/common"
	"github.com/ontio/ontology/smartcontract/states"
)

// BalanceOf calls a contract's balanceOf method for the given account.
func BalanceOf(acct common.Address, contract common.Address) (*big.Int, error) {
	r, err := Exec(contract, "balanceOf", []interface{}{acct})
	if err != nil {
		return nil, err
	}
	raw, ok := r.Result.(string)
	if !ok {
		return nil, fmt.Errorf(
			`chain: unexpected "balanceOf" response type: %s`,
			reflect.TypeOf(r),
		)
	}
	val, err := hex.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return common.BigIntFromNeoBytes(val), nil
}

// Exec executes a method on a contract with the given parameters.
func Exec(contract common.Address, method string, params []interface{}) (*states.PreExecResult, error) {
	mut, err := hcommon.NewNeovmInvokeTransaction(0, 0, contract, []interface{}{method, params})
	if err != nil {
		return nil, err
	}
	txn, err := mut.IntoImmutable()
	if err != nil {
		return nil, err
	}
	return ledger.DefLedger.PreExecuteContract(txn)
}

// NativeBalanceOf calls a contract's balanceOf method for the given account.
func NativeBalanceOf(acct common.Address, contract common.Address) (*big.Int, error) {
	r, err := NativeExec(contract, "balanceOf", []interface{}{acct[:]})
	if err != nil {
		return nil, err
	}
	raw, ok := r.Result.(string)
	if !ok {
		return nil, fmt.Errorf(
			`chain: unexpected "balanceOf" response type: %s`,
			reflect.TypeOf(r),
		)
	}
	val, err := hex.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return common.BigIntFromNeoBytes(val), nil
}

// NativeExec executes a method on a native contract with the given parameters.
func NativeExec(contract common.Address, method string, params []interface{}) (*states.PreExecResult, error) {
	mut, err := hcommon.NewNativeInvokeTransaction(0, 0, contract, 0, method, params)
	if err != nil {
		return nil, err
	}
	txn, err := mut.IntoImmutable()
	if err != nil {
		return nil, err
	}
	return ledger.DefLedger.PreExecuteContract(txn)
}
