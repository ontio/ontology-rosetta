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
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/ontio/ontology-rosetta/common"
	"github.com/ontio/ontology-rosetta/config"
	db "github.com/ontio/ontology-rosetta/store"
	util "github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/ledger"
	com "github.com/ontio/ontology/core/store/common"
	"github.com/ontio/ontology/errors"
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
		storeHeight, errmsg := getHeightFromStore(s.store)
		if errmsg != nil {
			return resp, STORE_DB_ERROR
		}
		if height > storeHeight {
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
	pageNum, err := store.GetData([]byte(key))
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		} else {
			return 0, nil
		}
	}
	page_num, err := strconv.ParseInt(string(pageNum), 10, 32)
	if err != nil {
		return 0, err
	}
	for i := page_num; i > 0; i-- {
		num := strconv.FormatInt(int64(i), 10)
		b, err := GetAccBalancesByPageNum(store, key, string(num))
		if err != nil {
			return 0, nil
		}
		if b.StartBlockNum > height {
			continue
		}
		sort.SliceStable(b.Value, func(i, j int) bool {
			if b.Value[i].Height >= b.Value[j].Height {
				return false
			}
			return true
		})
		for i := len(b.Value) - 1; i >= 0; i-- {
			if height >= b.Value[i].Height {
				return b.Value[i].Amount, nil
			}
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

func GetBlockHeight(store *db.Store, waitTime uint32) error {
	h, err := getHeightFromStore(store)
	if err != nil {
		return err
	}
	height := bactor.GetCurrentBlockHeight()
	go func() {
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
					if err == com.ErrNotFound {
						err := saveBlockHeight(store, i)
						if err != nil {
							notifyKillProcess()
							return
						}
						continue
					} else {
						log.RosetaaLog.Errorf("GetEventNotifyByHeight height:%d,err:%s", i, err.Error())
						notifyKillProcess()
						return
					}
				}
				if notify == nil {
					err = saveBlockHeight(store, i)
					if err != nil {
						notifyKillProcess()
						return
					}
					continue
				}
				transfers, err := parseEventNotify(notify, i)
				if err != nil {
					notifyKillProcess()
					return
				}
				err = dealTransferData(store, transfers, i)
				if err != nil {
					log.RosetaaLog.Errorf("dealTransferData height:%d,erp:%s", i, err)
					notifyKillProcess()
					return
				}
			}
			h = height
			height = bactor.GetCurrentBlockHeight()
			<-time.After(time.Second * time.Duration(waitTime))
		}
	}()
	return nil
}

func notifyKillProcess() {
	log.RosetaaLog.Info("notify kill rosetta process")
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	if err != nil {
		log.RosetaaLog.Fatalf("notifyKillProcess ,err:%s", err)
		panic(err)
	}
}

func parseEventNotify(execNotify []*event.ExecuteNotify, height uint32) ([]*transferInfo, error) {
	transfers := make([]*transferInfo, 0)
	for _, execute := range execNotify {
		for _, value := range execute.Notify {
			if value.States == nil {
				continue
			}
			contractAddress := value.ContractAddress.ToHexString()
			//skip when not a monitored contract
			if !util.IsMonitoredAddress(contractAddress) {
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
			transfer.contractAddr = value.ContractAddress.ToHexString()
			if value.ContractAddress.ToHexString() == util.ONT_ADDRESS || value.ContractAddress.ToHexString() == util.ONG_ADDRESS {
				method := slice.Index(0).Interface().(string)
				if method != config.OP_TYPE_TRANSFER {
					continue
				}
				fromAddr := slice.Index(1).Interface().(string)
				toAddr := slice.Index(2).Interface().(string)
				amount := slice.Index(3).Interface().(float64)
				if execute.State == event.CONTRACT_STATE_FAIL {
					if toAddr == util.GOVERNANCE_ADDR && value.ContractAddress.ToHexString() == util.ONG_ADDRESS {
						transfer.height = height
						transfer.fromAddr = fromAddr
						transfer.toAddr = toAddr
						transfer.amount = uint64(int(amount))
					} else {
						continue
					}
				} else {
					transfer.height = height
					transfer.fromAddr = fromAddr
					transfer.toAddr = toAddr
					transfer.amount = uint64(int(amount))
				}
			} else {
				method, err := common.HexToBytes(slice.Index(0).Interface().(string))
				if err != nil {
					log.RosetaaLog.Errorf("method HexToBytes height:%d err:%s", height, err)
					return nil, err
				}
				if !strings.EqualFold(string(method), config.OP_TYPE_TRANSFER) {
					continue
				}
				tmpAddr := slice.Index(1).Interface().(string)
				if tmpAddr != util.OPE4_ADDR_BASE {
					addFromTmp, err := common.HexToBytes(slice.Index(1).Interface().(string))
					if err != nil {
						log.RosetaaLog.Errorf("addFromTmp HexToBytes height:%d, err:%s", height, err)
						return nil, err
					}
					addFrom, err := common.AddressParseFromBytes(addFromTmp)
					if err != nil {
						log.RosetaaLog.Errorf("addFrom addrFrom parse addr height:%d,failed:%s", height, err)
						return nil, err
					}
					transfer.fromAddr = addFrom.ToBase58()
				} else {
					transfer.fromAddr = util.OPE4_ADDR_BASE
				}
				addrToTmp, err := common.HexToBytes(slice.Index(2).Interface().(string))
				if err != nil {
					log.RosetaaLog.Errorf("addrToTmp HexToBytes height:%d,err:%s", height, err)
					return nil, err
				}
				addrTo, err := common.AddressParseFromBytes(addrToTmp)
				if err != nil {
					log.RosetaaLog.Errorf("addrTo parse addr height:%d, failed:%s", height, err)
					return nil, err
				}
				transfer.toAddr = addrTo.ToBase58()
				tmp, err := common.HexToBytes(slice.Index(3).Interface().(string))
				if err != nil {
					log.RosetaaLog.Errorf("tmp HexToBytes height:%d,err:%s", height, err)
					return nil, err
				}
				amt := common.BigIntFromNeoBytes(tmp)
				amount := amt.Uint64()
				transfer.amount = amount
			}
			transfers = append(transfers, transfer)
		}
	}
	return transfers, nil
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

func (self *Balance) Serialization(sink *common.ZeroCopySink) {
	sink.WriteUint32(self.Height)
	sink.WriteUint64(self.Amount)
}

func (self *Balance) Deserialization(source *common.ZeroCopySource) error {
	h, eof := source.NextUint32()
	if eof {
		return errors.NewDetailErr(io.ErrUnexpectedEOF, errors.ErrNoCode, "serialization.ReadUint32, deserialize height error!")
	}
	a, eof := source.NextUint64()
	if eof {
		return errors.NewDetailErr(io.ErrUnexpectedEOF, errors.ErrNoCode, "serialization.ReadUint64, deserialize amount error!")
	}
	self.Height = h
	self.Amount = a
	return nil
}

type Balances struct {
	StartBlockNum uint32
	EndBlockNum   uint32
	Value         []*Balance `json:"value"`
}

func (self *Balances) Serialization() []byte {
	sink := common.NewZeroCopySink(nil)
	sink.WriteUint32(self.StartBlockNum)
	sink.WriteUint32(self.EndBlockNum)
	sink.WriteVarUint(uint64(len(self.Value)))
	for _, balance := range self.Value {
		s := common.NewZeroCopySink(nil)
		balance.Serialization(s)
		sink.WriteVarBytes(s.Bytes())
	}
	return sink.Bytes()
}

func (self *Balances) Deserialization(values []byte) error {
	source := common.NewZeroCopySource(values)
	startBlockNum, eof := source.NextUint32()
	if eof {
		return io.ErrUnexpectedEOF
	}
	self.StartBlockNum = startBlockNum

	endBlockNum, eof := source.NextUint32()
	if eof {
		return io.ErrUnexpectedEOF
	}
	self.EndBlockNum = endBlockNum

	n, _, irregular, eof := source.NextVarUint()
	if eof {
		return io.ErrUnexpectedEOF
	}
	if irregular {
		return common.ErrIrregularData
	}
	for i := 0; i < int(n); i++ {
		buf, _, irregular, eof := source.NextVarBytes()
		if eof {
			return io.ErrUnexpectedEOF
		}
		if irregular {
			return common.ErrIrregularData
		}
		s := common.NewZeroCopySource(buf)
		balance := &Balance{}
		err := balance.Deserialization(s)
		if err != nil {
			return err
		}
		self.Value = append(self.Value, balance)
	}
	return nil
}

type BalanceInfo struct {
	Key   string     `json:"key"`
	Value []*Balance `json:"value"`
}

func getAddrKey(addr, contractAddr string) string {
	return addr + ":" + contractAddr
}
func getAccountKey(addr, contractAddr, pageNum string) string {
	return addr + ":" + contractAddr + ":" + pageNum
}

func getAccKey(key, pageNum string) string {
	return key + ":" + pageNum
}

func getBlockHeightKey() []byte {
	return []byte("height")
}

func dealTransferData(store *db.Store, transfers []*transferInfo, height uint32) error {
	if len(transfers) == 0 {
		return nil
	}
	addrMap := make(map[string]*ValueInfo)
	for _, transfer := range transfers {
		if transfer.fromAddr != util.ONT_ADDR_BASE58 && transfer.fromAddr != util.OPE4_ADDR_BASE {
			fromKey := getAddrKey(transfer.fromAddr, transfer.contractAddr)
			if value, present := addrMap[fromKey]; !present {
				addrMap[fromKey] = &ValueInfo{
					height:    height,
					subAmount: transfer.amount,
				}
			} else {
				value.subAmount = value.subAmount + transfer.amount
			}
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
		pageNum, err := store.GetData([]byte(k))
		if err != nil {
			if err != leveldb.ErrNotFound {
				log.RosetaaLog.Errorf("getPageNum height:%d,k:%s,err:%s", height, k, err)
				return err
			} else {
				balances := make([]*Balance, 0)
				if v.addAmount < v.subAmount {
					log.RosetaaLog.Errorf("amount height:%d calcul err,addAmount:%d,subAmount:%d k:%s", height, v.addAmount, v.subAmount, k)
					return fmt.Errorf("amount calcul err,addAmount:%d,subAmount:%d", v.addAmount, v.subAmount)
				}
				balance := &Balance{
					Height: height,
					Amount: v.addAmount - v.subAmount,
				}
				balances = append(balances, balance)
				balanceInfos = append(balanceInfos, &BalanceInfo{
					Key:   k,
					Value: balances,
				})
			}
		} else {
			buf, err := store.GetData([]byte(getAddrKey(k, string(pageNum))))
			if err != nil {
				log.RosetaaLog.Errorf("GetData height:%d,err:%s", height, err)
				return err
			}
			b := &Balances{
				Value: make([]*Balance, 0),
			}
			err = b.Deserialization(buf)
			if err != nil {
				log.RosetaaLog.Errorf("Deserialization height:%d,err:%s", height, err)
				return err
			}
			sort.SliceStable(b.Value, func(i, j int) bool {
				if b.Value[i].Height >= b.Value[j].Height {
					return false
				}
				return true
			})
			if b.Value[len(b.Value)-1].Amount+v.addAmount < v.subAmount {
				log.RosetaaLog.Errorf("key:%s,current amount:%d,addAmount:%d,subAmount:%d,height:%d", k, b.Value[len(b.Value)-1].Amount, v.addAmount, v.subAmount, height)
				return fmt.Errorf("amount error")
			}
			var params []*Balance
			balance := &Balance{
				Height: height,
				Amount: b.Value[len(b.Value)-1].Amount + v.addAmount - v.subAmount,
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
		pageNum, err := GetAccountPage(store, balance.Key)
		if err != nil {
			return err
		}
		if pageNum == "" {
			if len(balance.Value)/util.EACH_PAGE_SVAE_BALANCE_NUM > 0 {
				pageNum := len(balance.Value) / util.EACH_PAGE_SVAE_BALANCE_NUM
				for i := 0; i <= pageNum; i++ {
					b := &Balances{
						Value: make([]*Balance, 0),
					}
					b.StartBlockNum = height
					b.EndBlockNum = height
					for k, v := range balance.Value {
						if k < i*util.EACH_PAGE_SVAE_BALANCE_NUM {
							continue
						}
						b.Value = append(b.Value, v)
						if len(b.Value) == util.EACH_PAGE_SVAE_BALANCE_NUM {
							break
						}
					}
					if len(b.Value) > 0 {
						buf := b.Serialization()
						pageNum := strconv.FormatInt(int64(i+1), 10)
						store.BatchPut([]byte(getAccKey(balance.Key, pageNum)), buf)
						store.BatchPut([]byte(balance.Key), []byte(pageNum))
					}
				}
			} else {
				b := &Balances{
					Value: make([]*Balance, 0),
				}
				b.StartBlockNum = height
				b.EndBlockNum = height
				for _, v := range balance.Value {
					b.Value = append(b.Value, v)
				}
				buf := b.Serialization()
				store.BatchPut([]byte(getAccKey(balance.Key, util.FIRST_PAGE_NUM)), buf)
				store.BatchPut([]byte(balance.Key), []byte(util.FIRST_PAGE_NUM))
			}
		} else {
			page_num, err := strconv.ParseInt(pageNum, 10, 32)
			if err != nil {
				return err
			}
			b, err := GetAccBalancesByPageNum(store, balance.Key, pageNum)
			if err != nil {
				log.RosetaaLog.Errorf("GetAccBalancesByPageNum height:%d,err:%s", height, err)
				return err
			}
			if (len(b.Value)+len(balance.Value))/util.EACH_PAGE_SVAE_BALANCE_NUM > 0 {
				b.EndBlockNum = height
				var index int
				for k, v := range balance.Value {
					b.Value = append(b.Value, v)
					index = k
					if len(b.Value) == util.EACH_PAGE_SVAE_BALANCE_NUM {
						break
					}
				}
				buf := b.Serialization()
				store.BatchPut([]byte(balance.Key), buf)
				pageNumber := (len(balance.Value) - index) / util.EACH_PAGE_SVAE_BALANCE_NUM
				for i := 0; i <= pageNumber; i++ {
					b := &Balances{
						Value: make([]*Balance, 0),
					}
					b.StartBlockNum = height
					b.EndBlockNum = height
					for k, v := range balance.Value {
						if k < i*util.EACH_PAGE_SVAE_BALANCE_NUM+index {
							continue
						}
						b.Value = append(b.Value, v)
					}
					if len(b.Value) > 0 {
						buf := b.Serialization()
						store.BatchPut([]byte(balance.Key), buf)
						pageNum := strconv.FormatInt(int64(i+1)+page_num, 10)
						store.BatchPut([]byte(getAccKey(balance.Key, pageNum)), buf)
						store.BatchPut([]byte(balance.Key), []byte(pageNum))
					}
				}
			} else {
				b.EndBlockNum = height
				for _, v := range balance.Value {
					b.Value = append(b.Value, v)
				}
				buf := b.Serialization()
				store.BatchPut([]byte(getAccKey(balance.Key, pageNum)), buf)
			}
		}
	}
	store.BatchPut(getBlockHeightKey(), []byte(strconv.FormatUint(uint64(height), 10)))
	err := store.CommitTo()
	if err != nil {
		log.RosetaaLog.Errorf("batchSaveBalance err:%s,height:%d", err, height)
		return err
	}
	return nil
}

func GetAccountPage(store *db.Store, key string) (string, error) {
	value, err := store.GetData([]byte(key))
	if err != nil {
		if err != leveldb.ErrNotFound {
			return "", err
		}
		return "", nil
	}
	return string(value), nil
}

func GetAccBalancesByPageNum(store *db.Store, key, pageNum string) (*Balances, error) {
	value, err := store.GetData([]byte(key + ":" + pageNum))
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	b := &Balances{
		Value: make([]*Balance, 0),
	}
	err = b.Deserialization(value)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func getHeightFromStore(store *db.Store) (uint32, error) {
	value, err := store.GetData(getBlockHeightKey())
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}
	height, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint32(height), nil
}

func saveBlockHeight(store *db.Store, height uint32) error {
	err := store.SaveData(getBlockHeightKey(), []byte(strconv.FormatUint(uint64(height), 10)))
	if err != nil {
		log.RosetaaLog.Error("SaveData err:%s,height:%d", err, height)
		return err
	}
	return nil
}
