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
	"fmt"
	"strings"

	rtypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-rosetta/config"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/http/base/actor"
	bcomn "github.com/ontio/ontology/http/base/common"
)

var (
	ONT_ADDRESS   = "0100000000000000000000000000000000000000"
	ONG_ADDRESS   = "0200000000000000000000000000000000000000"
	ONTID_ADDRESS = "0300000000000000000000000000000000000000"
	PARAM_ADDRESS = "0400000000000000000000000000000000000000"
	AUTH_ADDRESS  = "0600000000000000000000000000000000000000"
	GOV_ADDRESS   = "0700000000000000000000000000000000000000"

	ONT_ADDR_BASE58 = "AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM"
	GOVERNANCE_ADDR = "AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK"
	OPE4_ADDR_BASE  = "00"
	STATE_SUCCESS   = "SUCCESS"
	STATE_FAILED    = "FAILED"

	Currencies                 map[string]*rtypes.Currency
	EACH_PAGE_SVAE_BALANCE_NUM = 10
	FIRST_PAGE_NUM             = "1"
)

func InitCurrencies() error {
	Currencies = make(map[string]*rtypes.Currency)
	Currencies[ONT_ADDRESS] = &rtypes.Currency{
		Symbol:   constants.ONT_SYMBOL,
		Decimals: constants.ONT_DECIMALS,
		Metadata: GetMetatdata(ONT_ADDRESS),
	}
	Currencies[ONG_ADDRESS] = &rtypes.Currency{
		Symbol:   constants.ONG_SYMBOL,
		Decimals: constants.ONG_DECIMALS,
		Metadata: GetMetatdata(ONG_ADDRESS),
	}

	for _, scriptHash := range config.Conf.MonitorOEP4ScriptHash {
		symbol, err := GetSymbol(scriptHash)
		if err != nil {
			log.Debugf("get symbol from contract:%s ,failed:%s", scriptHash, err)
			continue
			//return fmt.Errorf("get symbol from contract:%s ,failed:%s", scriptHash, err)
		}
		decimal, err := GetDecimals(scriptHash)
		if err != nil {
			log.Debugf("get Decimals from contract:%s ,failed:%s", scriptHash, err)
			continue
		}
		metdata := GetMetatdata(scriptHash)
		Currencies[strings.ToLower(scriptHash)] = &rtypes.Currency{
			Symbol:   symbol,
			Decimals: decimal,
			Metadata: metdata,
		}
	}
	return nil
}

func TransformTransaction(tran *types.Transaction) (*rtypes.Transaction, error) {
	rt := new(rtypes.Transaction)
	tmphash := tran.Hash()
	rtIdentifier := &rtypes.TransactionIdentifier{Hash: tmphash.ToHexString()}
	rt.TransactionIdentifier = rtIdentifier
	opts := make([]*rtypes.Operation, 0)

	events, err := actor.GetEventNotifyByTxHash(tran.Hash())
	if err != nil {
		return nil, err
	}
	result := config.STATUS_SUCCESS.Status
	idx := 0
	for _, notify := range events.Notify {
		contractAddress := notify.ContractAddress.ToHexString()

		//skip when not a monitored contract
		if !IsMonitoredAddress(contractAddress) {
			continue
		}
		states := notify.States.([]interface{})
		//ONT and ONG
		if IsONT(contractAddress) || IsONG(contractAddress) {
			if len(states) == 4 { //['transfer',from,to,amount]

				//for genesis block
				//1. transfer 10^9 ONT from AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM (ONT address)
				//2. transfer 10^9 ONG(decimals 9) from  AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM to AFmseVrdL9f9oyCzZefL9tG6UbvhUMqNMV(Gov contract)
				//this will not raise as a operation

				if states[1].(string) != ONT_ADDR_BASE58 {
					//a transfer will divide into 2 operations
					//from and to
					//from operation
					fromOpt := new(rtypes.Operation)
					fromOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx * 2),
						//NetworkIndex: nil,
					}
					fromOpt.Status = result

					fromOpt.Type = config.OP_TYPE_TRANSFER // this should always be "transfer"
					amount := new(rtypes.Amount)
					//for from account ,the amount value should be "-" minus
					amount.Value = fmt.Sprintf("-%d", int64(states[3].(float64)))
					amount.Currency = GetCurrency(contractAddress)
					fromOpt.Amount = amount

					acct := new(rtypes.AccountIdentifier)
					acct.Address = states[1].(string)
					fromOpt.Account = acct

					//to operation
					toOpt := new(rtypes.Operation)
					toOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx*2 + 1),
						//NetworkIndex: nil,
					}
					toOpt.RelatedOperations = []*rtypes.OperationIdentifier{
						{
							Index: int64(idx * 2),
						},
					}
					toOpt.Type = config.OP_TYPE_TRANSFER
					toOpt.Status = result
					toAmount := new(rtypes.Amount)
					toAmount.Value = fmt.Sprintf("%d", int64(states[3].(float64)))
					toAmount.Currency = GetCurrency(contractAddress)
					toOpt.Amount = toAmount

					toAcct := new(rtypes.AccountIdentifier)
					toAcct.Address = states[2].(string)
					toOpt.Account = toAcct

					opts = append(opts, fromOpt, toOpt)
					idx += 1
				} else {
					//to operation
					toOpt := new(rtypes.Operation)
					toOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx * 2),
						//NetworkIndex: nil,
					}

					toOpt.Type = config.OP_TYPE_TRANSFER
					toOpt.Status = result
					toAmount := new(rtypes.Amount)
					toAmount.Value = fmt.Sprintf("%d", int64(states[3].(float64)))
					toAmount.Currency = GetCurrency(contractAddress)
					toOpt.Amount = toAmount

					toAcct := new(rtypes.AccountIdentifier)
					toAcct.Address = states[2].(string)
					toOpt.Account = toAcct

					opts = append(opts, toOpt)
					idx += 1
				}
			}
		} else {
			// deal oep4 token
			method, err := hex.DecodeString(states[0].(string))
			if err != nil {
				return nil, err
			}
			m := string(method)

			if len(states) == 4 && strings.EqualFold(m, config.OP_TYPE_TRANSFER) { //['transfer',from,to,amount]
				amtbytes, err := hex.DecodeString(states[3].(string))
				if err != nil {
					return nil, err
				}
				amt := common.BigIntFromNeoBytes(amtbytes)
				subacc := &rtypes.SubAccountIdentifier{Address: contractAddress}

				//all mint token transfer should ignore
				if len(states[1].(string)) > 0 && states[1].(string) != OPE4_ADDR_BASE {
					fromOpt := new(rtypes.Operation)
					fromOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx * 2),
						//NetworkIndex: nil,
					}
					fromOpt.Status = result
					fromOpt.Type = config.OP_TYPE_TRANSFER // this should always be "transfer"
					amount := new(rtypes.Amount)

					amount.Value = fmt.Sprintf("-%d", amt.Int64())
					amount.Currency = GetCurrency(contractAddress)
					fromOpt.Amount = amount
					addrFromTmp, _ := common.HexToBytes(states[1].(string))
					fromAcctAddr, err := common.AddressParseFromBytes(addrFromTmp)
					if err != nil {
						return nil, err
					}
					fromOpt.Account = &rtypes.AccountIdentifier{
						Address:    fromAcctAddr.ToBase58(),
						SubAccount: subacc,
					}
					//toOpt
					toOpt := new(rtypes.Operation)
					toOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx*2 + 1),
						//NetworkIndex: nil,
					}
					toOpt.RelatedOperations = []*rtypes.OperationIdentifier{
						{
							Index: int64(idx * 2),
						},
					}
					toOpt.Type = config.OP_TYPE_TRANSFER
					toOpt.Status = result
					toAmount := new(rtypes.Amount)
					toAmount.Value = fmt.Sprintf("%d", amt.Int64())
					toAmount.Currency = GetCurrency(contractAddress)
					toOpt.Amount = toAmount

					addrToTmp, _ := common.HexToBytes(states[2].(string))
					toAcctAddr, err := common.AddressParseFromBytes(addrToTmp)
					if err != nil {
						return nil, err
					}
					toOpt.Account = &rtypes.AccountIdentifier{
						Address:    toAcctAddr.ToBase58(),
						SubAccount: subacc,
					}
					opts = append(opts, fromOpt, toOpt)
					idx += 1
				} else {
					//toOpt
					toOpt := new(rtypes.Operation)
					toOpt.OperationIdentifier = &rtypes.OperationIdentifier{
						Index: int64(idx * 2),
						//NetworkIndex: nil,
					}

					toOpt.Type = config.OP_TYPE_TRANSFER
					toOpt.Status = result
					toAmount := new(rtypes.Amount)
					toAmount.Value = fmt.Sprintf("%d", amt.Int64())
					toAmount.Currency = GetCurrency(contractAddress)
					toOpt.Amount = toAmount

					addrToTmp, _ := common.HexToBytes(states[2].(string))
					toAcctAddr, err := common.AddressParseFromBytes(addrToTmp)
					if err != nil {
						return nil, err
					}
					toOpt.Account = &rtypes.AccountIdentifier{
						Address:    toAcctAddr.ToBase58(),
						SubAccount: subacc,
					}
					opts = append(opts, toOpt)
					idx += 1
				}
			}

		}

	}
	rt.Operations = opts
	return rt, nil
}

func IsONT(contractAddr string) bool {
	return strings.EqualFold(contractAddr, ONT_ADDRESS)
}

func IsONG(contractAddr string) bool {
	return strings.EqualFold(contractAddr, ONG_ADDRESS)
}

func isNative(contractAddr string) bool {
	return strings.EqualFold(contractAddr, ONT_ADDRESS) ||
		strings.EqualFold(contractAddr, ONG_ADDRESS) ||
		strings.EqualFold(contractAddr, ONTID_ADDRESS) ||
		strings.EqualFold(contractAddr, PARAM_ADDRESS) ||
		strings.EqualFold(contractAddr, AUTH_ADDRESS) ||
		strings.EqualFold(contractAddr, GOV_ADDRESS)
}

func IsOEP4(contractAddr string) bool {

	for _, oep4 := range config.Conf.MonitorOEP4ScriptHash {
		if strings.EqualFold(contractAddr, oep4) {
			return true
		}
	}
	return false
}

func GetDecimals(contractAddr string) (int32, error) {
	if IsONT(contractAddr) {
		return constants.ONT_DECIMALS, nil
	}
	if IsONG(contractAddr) {
		return constants.ONG_DECIMALS, nil
	}
	if IsOEP4(contractAddr) {
		r, err := PreExecNeovmContract(contractAddr, "decimals", nil)
		if err != nil {
			return -1, err
		}
		bs, err := hex.DecodeString(r.(string))
		if err != nil {
			return -1, err
		}
		return int32(common.BigIntFromNeoBytes(bs).Int64()), nil
	}
	return 0, fmt.Errorf("not a supported contract")
}

func GetSymbol(contractAddr string) (string, error) {
	if IsONT(contractAddr) {
		return constants.ONT_SYMBOL, nil
	}
	if IsONG(contractAddr) {
		return constants.ONG_SYMBOL, nil
	}
	if IsOEP4(contractAddr) {
		r, err := PreExecNeovmContract(contractAddr, "symbol", nil)
		if err != nil {
			return "", err
		}
		bs, err := hex.DecodeString(r.(string))
		if err != nil {
			return "", err
		}
		return string(bs), nil
	}

	return "UNKNOWN", fmt.Errorf("not a supported contract")
}

func GetMetatdata(contractAddr string) map[string]interface{} {
	meta := make(map[string]interface{})
	meta["ContractAddress"] = contractAddr
	if IsONT(contractAddr) {
		meta["TokenType"] = "Governance Token"
	}
	if IsONG(contractAddr) {
		meta["TokenType"] = "Utility Token"
	}
	if IsOEP4(contractAddr) {
		meta["TokenType"] = "OEP4 Token"
	}
	return meta
}

func GetCurrency(contractAddress string) *rtypes.Currency {
	c, ok := Currencies[strings.ToLower(contractAddress)]
	if !ok {

		symbol, err := GetSymbol(contractAddress)
		if err != nil {
			return nil
		}
		decimal, err := GetDecimals(contractAddress)
		if err != nil {
			return nil
		}

		currency := &rtypes.Currency{
			Symbol:   symbol,
			Decimals: decimal,
			Metadata: GetMetatdata(contractAddress),
		}
		Currencies[strings.ToLower(contractAddress)] = currency
		return currency
	}
	return c
}

func PreExecNeovmContract(contractAddress string, method string, params []interface{}) (interface{}, error) {

	addr, err := common.AddressFromHexString(contractAddress)
	if err != nil {
		return nil, err
	}
	mutTx, err := bcomn.NewNeovmInvokeTransaction(0, 0, addr, []interface{}{method, params})
	if err != nil {
		return nil, err
	}
	tx, err := mutTx.IntoImmutable()
	if err != nil {
		return nil, err
	}
	result, err := ledger.DefLedger.PreExecuteContract(tx)
	if err != nil {
		return nil, err
	}
	return result.Result, nil

}

func IsMonitoredAddress(contractAddress string) bool {
	return IsONT(contractAddress) || IsONG(contractAddress) || IsOEP4(contractAddress)
}
