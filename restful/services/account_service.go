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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	db "github.com/ontio/ontology-rosetta/store"
	util "github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	bactor "github.com/ontio/ontology/http/base/actor"
	bcomn "github.com/ontio/ontology/http/base/common"
	"github.com/ontio/ontology/smartcontract/event"
	"github.com/syndtr/goleveldb/leveldb"
)

type AccountAPIService struct {
	network *types.NetworkIdentifier
	store   *db.Store
}

func NewAccountAPIService(network *types.NetworkIdentifier, store *db.Store) server.AccountAPIServicer {
	return &AccountAPIService{
		network: network,
		store:   store,
	}
}

func (s *AccountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	resp := &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{},
		Balances:        make([]*types.Amount, 0),
		Metadata:        make(map[string]interface{}),
	}
	if request.AccountIdentifier == nil {
		return resp, PARAMS_ERROR
	}
	address, err := common.AddressFromBase58(request.AccountIdentifier.Address)
	if err != nil {
		return resp, ADDRESS_INVALID
	}
	if request.BlockIdentifier == nil {
		if request.AccountIdentifier.SubAccount == nil {
			return getCurrentOntOngBalance(resp, address)
		} else {
			//oep4 balance
			return getCurrentOep4Balance(resp, address, request.AccountIdentifier.SubAccount.Address)
		}
	} else {
		height, err := getBlockIdentifierHeight(request.BlockIdentifier)
		if err != nil {
			return resp, err
		}
		if height > getHeightFromStore(s.store) {
			return resp, HEIGHT_HISTORICAL_LESS_THAN_CURRENT
		}
		if request.AccountIdentifier.SubAccount == nil {
			return getHistoricalOntOngBalance(resp, s.store, height, request.AccountIdentifier.Address)
		} else {
			//ope4 balance
			return getHistoricalOep4Balance(resp, s.store, height, request.AccountIdentifier.Address, request.AccountIdentifier.SubAccount.Address)
		}
	}
}

func getCurrentOntOngBalance(resp *types.AccountBalanceResponse, address common.Address) (*types.AccountBalanceResponse, *types.Error) {
	balance, err := bcomn.GetBalance(address)
	if err != nil {
		return resp, BALANCE_ERROR
	}
	index, err := strconv.ParseInt(balance.Height, 10, 64)
	if err != nil {
		return resp, PARSE_INT_ERROR
	}
	//block height
	resp.BlockIdentifier.Index = index
	//block hash
	hash := bactor.GetBlockHashFromStore(uint32(index))
	if hash == common.UINT256_EMPTY {
		return resp, BLOCK_HASH_INVALID
	}
	resp.BlockIdentifier.Hash = hash.ToHexString()
	amounts := make([]*types.Amount, 0)
	amounts = append(amounts, getCurrencyAmount(balance.Ont, util.ONT_ADDRESS))
	amounts = append(amounts, getCurrencyAmount(balance.Ong, util.ONG_ADDRESS))
	resp.Balances = amounts
	return resp, nil
}

func getCurrentOep4Balance(resp *types.AccountBalanceResponse, address common.Address, oep4ContractAddr string) (*types.AccountBalanceResponse, *types.Error) {
	//oep4 balance
	if !util.IsOEP4(oep4ContractAddr) {
		return resp, CONTRACT_ADDRESS_ERROR
	}
	contractAddr, err := common.AddressFromHexString(oep4ContractAddr)
	if err != nil {
		return resp, CONTRACT_ADDRESS_ERROR
	}
	resp.BlockIdentifier.Index = int64(bactor.GetCurrentBlockHeight())
	hash := bactor.CurrentBlockHash()
	resp.BlockIdentifier.Hash = hash.ToHexString()
	mutTx, err := bcomn.NewNeovmInvokeTransaction(0, 0, contractAddr, []interface{}{"balanceOf", []interface{}{address}})
	if err != nil {
		return resp, TXHASH_INVALID
	}
	tx, err := mutTx.IntoImmutable()
	if err != nil {
		return resp, TXHASH_INVALID
	}
	result, err := ledger.DefLedger.PreExecuteContract(tx)
	if err != nil {
		return resp, PRE_EXECUTE_ERROR
	}
	value, err := hex.DecodeString(result.Result.(string))
	if err != nil {
		return resp, PRE_EXECUTE_ERROR
	}
	amt := common.BigIntFromNeoBytes(value)
	amounts := make([]*types.Amount, 0)
	amounts = append(amounts, getCurrencyAmount(fmt.Sprintf("%d", amt.Int64()), oep4ContractAddr))
	resp.Balances = amounts
	return resp, nil
}

func getHistoricalOntOngBalance(resp *types.AccountBalanceResponse, store *db.Store, height uint32, addr string) (*types.AccountBalanceResponse, *types.Error) {
	ontBalance, err := getHistoryBalance(store, height, addr, util.ONT_ADDRESS)
	if err != nil {
		return resp, QUERY_BALANCE_ERROR
	}
	ongBalance, err := getHistoryBalance(store, height, addr, util.ONG_ADDRESS)
	if err != nil {
		return resp, QUERY_BALANCE_ERROR
	}
	resp.BlockIdentifier.Index = int64(height)
	hash := bactor.GetBlockHashFromStore(height)
	resp.BlockIdentifier.Hash = hash.ToHexString()
	amounts := make([]*types.Amount, 0)
	amounts = append(amounts, getCurrencyAmount(strconv.FormatUint(ontBalance, 10), util.ONT_ADDRESS))
	amounts = append(amounts, getCurrencyAmount(strconv.FormatUint(ongBalance, 10), util.ONG_ADDRESS))
	resp.Balances = amounts
	return resp, nil
}

func getHistoricalOep4Balance(resp *types.AccountBalanceResponse, store *db.Store, height uint32, addr, ope4ContractAddr string) (*types.AccountBalanceResponse, *types.Error) {
	oep4Balance, err := getHistoryBalance(store, height, addr, ope4ContractAddr)
	if err != nil {
		return resp, QUERY_BALANCE_ERROR
	}
	resp.BlockIdentifier.Index = int64(height)
	hash := bactor.GetBlockHashFromStore(height)
	resp.BlockIdentifier.Hash = hash.ToHexString()
	amounts := make([]*types.Amount, 0)
	amounts = append(amounts, getCurrencyAmount(strconv.FormatUint(oep4Balance, 10), ope4ContractAddr))
	resp.Balances = amounts
	return resp, nil
}

func getCurrencyAmount(balance string, contractAddr string) *types.Amount {
	amount := &types.Amount{
		Currency: &types.Currency{},
		Metadata: make(map[string]interface{}),
	}
	amount.Value = balance
	amount.Currency = util.GetCurrency(contractAddr)
	return amount
}

func getBlockIdentifierHeight(blockIdentifier *types.PartialBlockIdentifier) (uint32, *types.Error) {
	var height uint32
	if blockIdentifier.Hash != nil {
		hash, err := common.Uint256FromHexString(*blockIdentifier.Hash)
		if err != nil {
			return 0, BLOCK_HASH_INVALID
		}
		block, err := bactor.GetBlockFromStore(hash)
		if err != nil {
			return 0, GET_BLOCK_FAILED
		}
		if block == nil {
			return 0, UNKNOWN_BLOCK
		}
		if block.Header == nil {
			return 0, UNKNOWN_BLOCK
		}
		height = block.Header.Height
	} else {
		height = uint32(*blockIdentifier.Index)
	}
	return height, nil
}

func getHistoryBalance(store *db.Store, height uint32, addr, contract string) (uint64, error) {
	key := getAddrKey(addr, contract)
	value, err := store.GetData([]byte(key))
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		} else {
			return 0, nil
		}
	}
	var balances []*Balance
	err = json.Unmarshal(value, &balances)
	if err != nil {
		return 0, err
	}
	sort.SliceStable(balances, func(i, j int) bool {
		if balances[i].Height > balances[j].Height {
			return false
		}
		return true
	})
	for i := len(balances) - 1; i >= 0; i-- {
		if height >= balances[i].Height {
			return balances[i].Amount, nil
		}
	}
	return 0, nil
}

type transferInfo struct {
	fromAddr     string
	toAddr       string
	amount       uint64
	contractAddr string
	height       uint32
}

func GetBlockHeight(store *db.Store) {
	h := getHeightFromStore(store)
	height := bactor.GetCurrentBlockHeight()
	for {
		var num uint32
		if h == 0 {
			num = 0
		} else {
			num = h + 1
		}
		for i := num; i <= height; i++ {
			notify, err := bactor.GetEventNotifyByHeight(i)
			if err != nil {
				if err.Error() == "not found" {
					saveBlockHeight(store, i)
					continue
				} else {
					log.Errorf("GetEventNotifyByHeight height:%d,err:%s", i, err.Error())
					panic(err)
				}
			}
			if notify == nil {
				saveBlockHeight(store, i)
				continue
			}
			transfers := parseEventNotify(notify, i)
			err = dealTransferData(store, transfers, i)
			if err != nil {
				log.Errorf("err:%s", err)
				panic(err)
			}
		}
		h = height
		height = bactor.GetCurrentBlockHeight()
		<-time.After(time.Second * 20)
	}
}

func parseEventNotify(execNotify []*event.ExecuteNotify, height uint32) []*transferInfo {
	transfers := make([]*transferInfo, 0)
	for _, execute := range execNotify {
		if execute.State == event.CONTRACT_STATE_FAIL {
			continue
		}
		for _, value := range execute.Notify {
			if value.States == nil {
				continue
			}
			if reflect.TypeOf(value.States).Kind() != reflect.Slice {
				continue
			}
			slice := reflect.Indirect(reflect.ValueOf(value.States))
			if slice.Len() != 4 {
				continue
			}
			transfer := &transferInfo{}
			transfer.height = height
			transfer.contractAddr = value.ContractAddress.ToHexString()
			if value.ContractAddress.ToHexString() == util.ONT_ADDRESS || value.ContractAddress.ToHexString() == util.ONG_ADDRESS {
				method := slice.Index(0).Interface().(string)
				if method != "transfer" {
					continue
				}
				transfer.fromAddr = slice.Index(1).Interface().(string)
				transfer.toAddr = slice.Index(2).Interface().(string)
				amount := slice.Index(3).Interface().(float64)
				if value.ContractAddress.ToHexString() == util.ONT_ADDRESS {
					coinAmount := strconv.FormatFloat(amount, 'f', 0, 64)
					value, err := strconv.ParseUint(coinAmount, 10, 64)
					if err != nil {
						log.Errorf("ont parse value height:%d err:%s", height, err)
						panic(err)
					}
					transfer.amount = value
				} else if value.ContractAddress.ToHexString() == util.ONG_ADDRESS {
					coinAmount := strconv.FormatFloat(amount, 'f', 9, 64)
					coinamount := strings.Split(coinAmount, ".")
					if len(coinamount) > 1 {
						coinAmount = coinamount[0]
					}
					value, err := strconv.ParseUint(coinAmount, 10, 64)
					if err != nil {
						log.Errorf("ong parse value height:%d err:%s", height, err)
						panic(err)
					}
					transfer.amount = value
				}
			} else {
				method, err := common.HexToBytes(slice.Index(0).Interface().(string))
				if err != nil {
					log.Errorf("method HexToBytes err:%s", err)
					panic(err)
				}
				if string(method) != "transfer" {
					continue
				}
				addFromTmp, err := common.HexToBytes(slice.Index(1).Interface().(string))
				if err != nil {
					log.Errorf("addFromTmp HexToBytes err:%s", err)
					panic(err)
				}
				addFrom, err := common.AddressParseFromBytes(addFromTmp)
				if err != nil {
					log.Errorf("addFrom addrFrom parse addr failed:%s", err)
					panic(err)
				}
				transfer.fromAddr = addFrom.ToBase58()

				addrToTmp, err := common.HexToBytes(slice.Index(2).Interface().(string))
				if err != nil {
					log.Errorf("addrToTmp HexToBytes err:%s", err)
					panic(err)
				}
				addrTo, err := common.AddressParseFromBytes(addrToTmp)
				if err != nil {
					log.Errorf("addrTo parse addr failed:%s", err)
					panic(err)
				}
				transfer.toAddr = addrTo.ToBase58()
				tmp, err := common.HexToBytes(slice.Index(3).Interface().(string))
				if err != nil {
					log.Errorf("tmp HexToBytes err:%s", err)
					panic(err)
				}
				amt := common.BigIntFromNeoBytes(tmp)
				amount := amt.Uint64()
				transfer.amount = amount
			}
			transfers = append(transfers, transfer)
		}
	}
	return transfers
}

type ValueInfo struct {
	height    uint32
	subAmount uint64
	addAmount uint64
}

type Balance struct {
	Height uint32 `json:"height"`
	Amount uint64 `json:"amount"`
}
type BalanceInfo struct {
	Key   string     `json:"key"`
	Value []*Balance `json:"value"`
}

func getAddrKey(addr, contractAddr string) string {
	return addr + ":" + contractAddr
}

func getBlockHeightKey() []byte {
	return []byte("height")
}

func dealTransferData(store *db.Store, tranfers []*transferInfo, height uint32) error {
	addrMap := make(map[string]*ValueInfo)
	for _, transfer := range tranfers {
		fromKey := getAddrKey(transfer.fromAddr, transfer.contractAddr)
		if value, present := addrMap[fromKey]; !present {
			addrMap[fromKey] = &ValueInfo{
				height:    height,
				subAmount: transfer.amount,
			}
		} else {
			value.subAmount = value.subAmount + transfer.amount
		}

		toKey := getAddrKey(transfer.toAddr, transfer.contractAddr)
		if value, present := addrMap[toKey]; !present {
			addrMap[toKey] = &ValueInfo{
				height:    height,
				addAmount: transfer.amount,
			}
		} else {
			value.addAmount = value.addAmount + transfer.amount
		}
	}

	balanceInfos := make([]*BalanceInfo, 0)
	for k, v := range addrMap {
		value, err := store.GetData([]byte(k))
		if err != nil {
			if err != leveldb.ErrNotFound {
				log.Error(err)
				panic(err)
			} else {
				balances := make([]*Balance, 0)
				balance := &Balance{
					Height: height,
					Amount: v.addAmount - v.subAmount, //need check
				}
				balances = append(balances, balance)
				balanceInfos = append(balanceInfos, &BalanceInfo{
					Key:   k,
					Value: balances,
				})
			}
		} else {
			var params []*Balance
			err = json.Unmarshal(value, &params)
			if err != nil {
				panic(err)
			}
			sort.SliceStable(params, func(i, j int) bool {
				if params[i].Height > params[j].Height {
					return false
				}
				return true
			})
			if params[len(params)-1].Amount+v.addAmount < v.subAmount {
				return fmt.Errorf("amount error")
			}
			balance := &Balance{
				Height: height,
				Amount: params[len(params)-1].Amount + v.addAmount - v.subAmount,
			}
			params = append(params, balance)
			balanceInfos = append(balanceInfos, &BalanceInfo{
				Key:   k,
				Value: params,
			})
		}
	}
	return batchSaveBalance(store, height, balanceInfos)
}

func batchSaveBalance(store *db.Store, height uint32, balances []*BalanceInfo) error {
	store.NewBatch()
	for _, balance := range balances {
		buf, err := json.Marshal(balance.Value)
		if err != nil {
			log.Errorf("unmarshal err:%s", err)
			panic(err)
		}
		store.BatchPut([]byte(balance.Key), buf)
	}
	store.BatchPut(getBlockHeightKey(), []byte(strconv.FormatUint(uint64(height), 10)))
	err := store.CommitTo()
	if err != nil {
		log.Errorf("batchSaveBalance err:%s", err)
		panic(err)
	}
	return nil
}

func getHeightFromStore(store *db.Store) uint32 {
	value, err := store.GetData(getBlockHeightKey())
	if err != nil {
		if err != leveldb.ErrNotFound {
			panic(err)
		}
		return 0
	}
	height, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		panic(err)
	}
	return uint32(height)
}

func saveBlockHeight(store *db.Store, height uint32) {
	err := store.SaveData(getBlockHeightKey(), []byte(strconv.FormatUint(uint64(height), 10)))
	if err != nil {
		log.Error(err)
		panic(err)
	}
}
