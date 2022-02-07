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
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcom "github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ontio/ontology-rosetta/chain"
	"github.com/ontio/ontology-rosetta/lexinum"
	"github.com/ontio/ontology-rosetta/log"
	"github.com/ontio/ontology-rosetta/model"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/constants"
	store "github.com/ontio/ontology/core/store/common"
	"github.com/ontio/ontology/core/store/ledgerstore"
	ctypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/http/base/actor"
	"github.com/ontio/ontology/smartcontract/event"
	"google.golang.org/protobuf/proto"
)

var (
	jsonNumberType = reflect.TypeOf(json.Number(""))
)

// NOTE(tav): We store the blockchain data within Badger using the following
// key/value structure:
//
//          accountKey a<acct><contract><height-lexinum> = <amount-bytes>
//            blockKey b<height-little-endian> = Block
// blockHash2HeightKey c<block-hash> = <height-little-endian>
// blockHeight2HashKey d<height-little-endian> = <block-hash>
//          txnHashKey e<unsigned-txn-hash> = <nil>
//                     height = <height-little-endian>
//
// We compress some of the native contract addresses, e.g. ONT/ONG, to single
// bytes so as to reduce space usage. An additional byte is used to indicate
// whether the address has been compressed.
//
// The default MemTableSize of 64MB seems to be large enough for the data we
// need to write in a single transaction while indexing a block. If this proves
// insufficient in the future, we can increase the size, or break up the
// transaction into smaller atomic units.

// Store aggregates the blockchain data for Rosetta API calls.
type Store struct {
	db            *badger.DB
	heightIndexed *int64
	heightSynced  *int64
	mu            sync.RWMutex // protects heightIndex, heightSynced
	tokens        map[common.Address]*currencyInfo
	parsedAbi     abi.ABI
}

// Close closes the store database. It must be called to ensure all pending
// writes are written to disk.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	log.Info("Closing internal data store")
	return s.db.Close()
}

// IndexBlocks polls the node for new blocks and indexes the block data.
func (s *Store) IndexBlocks(ctx context.Context, cfg IndexConfig) {
	const debug = 0
outer:
	for {
		time.Sleep(cfg.WaitTime)
		select {
		case <-ctx.Done():
			cfg.Done <- true
			return
		default:
		}
		height := s.getHeight()
		if height > 0 {
			height++
		}
		latest := actor.GetCurrentBlockHeight()
		if cfg.ExitEarly && height == latest+1 {
			return
		}
		for ; height <= latest; height++ {
			select {
			case <-ctx.Done():
				cfg.Done <- true
				return
			default:
			}
			if height%100 == 0 {
				log.Infof("Indexing block at height %d", height)
			}
			src, err := actor.GetBlockByHeight(height)
			if err != nil {
				log.Errorf("Failed to get block at height %d: %s", height, err)
				continue outer
			}
			var (
				changes []*balanceChange
				hashes  [][]byte
			)
			diffs := map[common.Address]map[common.Address]*big.Int{}
			henc := lexinum.EncodeHeight(height)
			hlen := len(henc)
			id := &blockID{
				hash:   src.Hash(),
				height: height,
			}
			dst := &model.Block{
				Timestamp: src.Header.Timestamp,
			}
			offsets := map[common.Uint256]int{}
			for i, txn := range src.Transactions {
				// NOTE(tav): We compute the unsigned transaction hash so that
				// we can detect potential conflicts when generating nonces.
				mut := &ctypes.MutableTransaction{
					GasLimit: txn.GasLimit,
					GasPrice: txn.GasPrice,
					Nonce:    txn.Nonce,
					Payer:    txn.Payer,
					Payload:  txn.Payload,
					TxType:   txn.TxType,
					Version:  txn.Version,
				}
				mhash := mut.Hash()
				hashes = append(hashes, mhash[:])
				hash := txn.Hash()
				dst.Transactions = append(dst.Transactions, &model.Transaction{
					Hash: hash[:],
				})
				offsets[hash] = i
			}
			evts, err := actor.GetEventNotifyByHeight(height)
			if err != nil {
				if err != store.ErrNotFound {
					log.Fatalf("Failed to get events at height %d: %s", height, err)
				}
				goto done
			}
			if evts == nil {
				goto done
			}
			for _, info := range evts {
				failed := info.State == event.CONTRACT_STATE_FAIL
				gasVerified := false
				offset := offsets[info.TxHash]
				ori := src.Transactions[offset]
				txn := dst.Transactions[offset]
				txn.Failed = failed
				for _, evt := range info.Notify {
					_, ok := s.tokens[evt.ContractAddress]
					if !ok {
						continue
					}
					//check evm ong event log
					var xfer *transfer
					isEvm, eventLog := checkEvmEventLog(evt)
					if isEvm {
						xfer, err = parseEvmOngTransferLog(eventLog, s.parsedAbi, info.GasConsumed)
						if err != nil {
							log.Warnf("parse evm ong err:%s,height:%d,txhash:%s", err, height, info.TxHash.ToHexString())
							continue
						}
					} else {
						xfer = decodeTransfer(height, info, evt)
						if xfer == nil {
							log.Warnf(
								"No transfer detected for state %#v in transaction %s at height %d",
								evt.States, info.TxHash.ToHexString(), height)
							continue
						}
					}
					gasverified, isgas, isContinue := checkgasVerified(evt.ContractAddress, ori.Payer, xfer.from, failed, gasVerified, xfer.isGas)
					if isContinue {
						continue
					}
					gasVerified = gasverified
					xfer.isGas = isgas
					txn.Transfers = append(txn.Transfers, balanceCal(xfer, evt, diffs))
				}
				// NOTE(tav): We log the cases where a transfer event wasn't
				// emitted for used gas.
				if info.GasConsumed != 0 && !gasVerified {
					log.Warnf(
						"Missing gas fee transfer event for txn %s at height %d",
						info.TxHash.ToHexString(), height,
					)
				}
			}
			// Encode db keys for account balance changes.
			for addr, accts := range diffs {
				base := append([]byte{'a'}, addr2slice(addr)...)
				lbase := len(base)
				for contract, diff := range accts {
					caddr := addr2slice(contract)
					prefix := make([]byte, lbase+len(caddr))
					n := copy(prefix, base)
					copy(prefix[n:], caddr)
					key := make([]byte, len(prefix)+hlen)
					n = copy(key, prefix)
					copy(key[n:], henc)
					changes = append(changes, &balanceChange{
						diff:   diff,
						key:    key,
						prefix: prefix,
					})
				}
			}
		done:
			switch debug {
			case 1:
				for _, txn := range dst.Transactions {
					if len(txn.Transfers) > 0 {
						log.Infof("Indexing transaction %s at height %d", txn, height)
					}
				}
			case 2:
				log.Infof("Saving %s with %s at height %d", id, dst, height)
			}
			err = s.setBlock(&blockState{
				block:   dst,
				changes: changes,
				hashes:  hashes,
				id:      id,
				synced:  latest,
			})
			if err != nil {
				log.Errorf("Failed to store block at height %d: %s", height, err)
				continue outer
			}
		}
	}
}

func (s *Store) Validate() {
	height := s.getHeight()
	latest := actor.GetCurrentBlockHeight()
	if height != latest {
		log.Fatalf("Indexed height %d does not match latest synced block %d", height, latest)
	}
	log.Infof("Validating store at block height %d", height)
	var accts []accountInfo
	total := 0
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1000
		opts.Prefix = []byte("a")
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()
		var ident []byte
		it.Seek([]byte("b"))
		log.Info("Finding unique account/contract combinations")
		for ; it.Valid(); it.Next() {
			total++
			key := it.Item().Key()
			start := -1
			switch key[1] {
			case 1:
				start = 3
			case 0:
				start = 22
			default:
				return fmt.Errorf("invalid account key found: %q", string(key))
			}
			end := -1
			switch key[start] {
			case 1:
				end = start + 2
			case 0:
				end = start + 21
			default:
				return fmt.Errorf("invalid account key found: %q", string(key))
			}
			if bytes.Equal(key[:end], ident) {
				continue
			}
			ident = make([]byte, end)
			copy(ident, key)
			ori := make([]byte, len(key))
			copy(ori, key)
			accts = append(accts, accountInfo{
				acct:     decompressAddr(key[1:start]),
				contract: decompressAddr(key[start:end]),
				key:      ori,
				native:   key[start] == 1,
			})
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Unable to calculate number of unique accounts: %s", err)
	}
	log.Infof("Found %d unique account/contract combinations out of %d", len(accts), total)
	err = s.db.View(func(txn *badger.Txn) error {
		var (
			balance *big.Int
			err     error
		)
		for i, info := range accts {
			if i%100 == 0 {
				log.Infof("Validated %d balances of %d", i, len(accts))
			}
			if info.native {
				balance, err = chain.NativeBalanceOf(info.acct, info.contract)
			} else {
				balance, err = chain.BalanceOf(info.acct, info.contract)
			}
			if err != nil {
				return fmt.Errorf(
					"unable to get balanceOf account %s for %s (%d): %s",
					info.acct.ToBase58(), info.contract.ToHexString(), i, err,
				)
			}
			item, err := txn.Get(info.key)
			if err != nil {
				return fmt.Errorf(
					"unable to get stored balance of account %s for %s (%d): %s",
					info.acct.ToBase58(), info.contract.ToHexString(), i, err,
				)
			}
			err = item.Value(func(val []byte) error {
				amount := new(big.Int).SetBytes(val)
				if amount.Cmp(balance) != 0 {
					return fmt.Errorf(
						"balance of account %s for %s (%d) does not match: stored (%s), on chain (%s)",
						info.acct.ToBase58(), info.contract.ToHexString(), i, amount, balance,
					)
				}
				return nil
			})
			if err != nil {
				if info.native {
					return err
				}
				log.Warnf("Validation failed for non-native OEP4 token: %s", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Unable to validate account balances: %s", err)
	}
	log.Infof("Successfully validated all balances")
}

func (s *Store) checkUnsignedTxHash(hash common.Uint256) (bool, *types.Error) {
	key := make([]byte, 33)
	key[0] = 'e'
	copy(key[1:], hash[:])
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, nil
		}
		return false, wrapErr(errDatastore, err)
	}
	return true, nil
}

func (s *Store) getBalance(
	pid *types.PartialBlockIdentifier,
	acct common.Address,
	currencies []*types.Currency,
	contracts ...common.Address,
) (*types.AccountBalanceResponse, *types.Error) {
	info, xerr := s.getBlockInfo(pid, false)
	if xerr != nil {
		return nil, xerr
	}
	balances := []*types.Amount{}
	filter := map[*types.Currency]bool{}
	for _, currency := range currencies {
		cinfo, xerr := s.validateCurrency(currency)
		if xerr != nil {
			return nil, xerr
		}
		filter[cinfo.currency] = true
	}
	for _, contract := range contracts {
		cinfo, xerr := s.getCurrencyInfo(contract)
		if xerr != nil {
			return nil, xerr
		}
		if len(currencies) > 0 {
			if !filter[cinfo.currency] {
				continue
			}
		}
		balance := &big.Int{}
		prefix := accountKeyPrefix(addr2slice(acct), addr2slice(contract))
		key := make([]byte, len(prefix)+len(info.hval))
		n := copy(key, prefix)
		copy(key[n:], info.hval)
		err := s.db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.IteratorOptions{
				Reverse: true,
			})
			defer it.Close()
			it.Seek(key)
			if !it.ValidForPrefix(prefix) {
				return nil
			}
			item := it.Item()
			return item.Value(func(val []byte) error {
				balance.SetBytes(val)
				return nil
			})
		})
		if err != nil {
			log.Errorf(
				"Unexpected error fetching %s balance for %s at %s from store: %s",
				contract.ToHexString(), acct.ToBase58(), info.height, err,
			)
			return nil, wrapErr(errDatastore, err)
		}
		balances = append(balances, &types.Amount{
			Currency: cinfo.currency,
			Value:    balance.String(),
		})
	}
	return &types.AccountBalanceResponse{
		Balances:        balances,
		BlockIdentifier: info.blockID,
	}, nil
}

func (s *Store) getBlockID(pid *types.PartialBlockIdentifier, nullable bool) (*blockID, *types.Error) {
	if nullable {
		if pid == nil || (pid.Hash == nil && pid.Index == nil) {
			return &blockID{
				byHeight: true,
				height:   s.getHeight(),
			}, nil
		}
	}
	if pid == nil || (pid.Hash == nil && pid.Index == nil) {
		return nil, errInvalidBlockIdentifier
	}
	id := &blockID{}
	if pid.Hash != nil {
		hash, err := common.Uint256FromHexString(*pid.Hash)
		if err != nil {
			return nil, errInvalidBlockHash
		}
		id.hash = hash
	}
	if pid.Index != nil {
		idx := *pid.Index
		if idx < 0 || idx > math.MaxUint32 {
			return nil, errInvalidBlockIndex
		}
		id.byHeight = true
		id.height = uint32(idx)
	}
	return id, nil
}

func (s *Store) getBlockInfo(pid *types.PartialBlockIdentifier, withBlock bool) (*blockInfo, *types.Error) {
	id, xerr := s.getBlockID(pid, true)
	if xerr != nil {
		return nil, xerr
	}
	latest := s.getHeight()
	if id.byHeight && id.height > latest {
		return nil, errUnknownBlockIndex
	}
	return s.getBlockInfoRaw(id, withBlock)
}

func (s *Store) getBlockInfoRaw(id *blockID, withBlock bool) (*blockInfo, *types.Error) {
	var xerr *types.Error
	block := &model.Block{}
	err := s.db.View(func(txn *badger.Txn) error {
		if id.byHeight {
			item, err := txn.Get(blockHeight2HashKey(id.height))
			if err != nil {
				switch err {
				case badger.ErrConflict:
					xerr = errDatastoreConflict
					return nil
				case badger.ErrKeyNotFound:
					xerr = errUnknownBlockIndex
					return nil
				}
				return err
			}
			err = item.Value(func(val []byte) error {
				if id.hash != common.UINT256_EMPTY {
					hash := common.Uint256{}
					copy(hash[:], val)
					if id.hash != hash {
						xerr = errInvalidBlockIdentifier
						return nil
					}
				}
				copy(id.hash[:], val)
				return nil
			})
			if err != nil {
				if err == badger.ErrConflict {
					xerr = errDatastoreConflict
					return nil
				}
				return err
			}
		} else {
			item, err := txn.Get(blockHash2HeightKey(id.hash[:]))
			if err != nil {
				switch err {
				case badger.ErrConflict:
					xerr = errDatastoreConflict
					return nil
				case badger.ErrKeyNotFound:
					xerr = errUnknownBlockHash
					return nil
				}
				return err
			}
			err = item.Value(func(val []byte) error {
				height := binary.LittleEndian.Uint32(val)
				if id.byHeight && id.height != height {
					xerr = errInvalidBlockIdentifier
					return nil
				}
				id.height = height
				return nil
			})
			if err != nil {
				if err == badger.ErrConflict {
					xerr = errDatastoreConflict
					return nil
				}
				return err
			}
		}
		if !withBlock {
			return nil
		}
		item, err := txn.Get(blockKey(id.height))
		if err != nil {
			if err == badger.ErrConflict {
				xerr = errDatastoreConflict
				return nil
			}
			return err
		}
		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, block)
		})
	})
	if xerr != nil {
		return nil, xerr
	}
	if err != nil {
		log.Errorf("Unexpected error fetching %s from store: %s", id, err)
		return nil, wrapErr(errDatastore, err)
	}
	return &blockInfo{
		block: block,
		blockID: &types.BlockIdentifier{
			Hash:  id.hash.ToHexString(),
			Index: int64(id.height),
		},
		height: id.height,
		hval:   lexinum.EncodeHeight(id.height),
	}, nil
}

func (s *Store) getCurrencyInfo(addr common.Address) (*currencyInfo, *types.Error) {
	c, ok := s.tokens[addr]
	if !ok {
		return c, errCurrencyNotDefined
	}
	return c, nil
}

func (s *Store) getHeight() uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.heightIndexed == nil {
		return 0
	}
	return uint32(*s.heightIndexed)
}

func (s *Store) setBlock(state *blockState) error {
	blockKey := blockKey(state.id.height)
	blockData, err := proto.Marshal(state.block)
	if err != nil {
		return fmt.Errorf("services: failed to encode model.Block: %s", err)
	}
	hashKey := blockHash2HeightKey(state.id.hash[:])
	heightKey := blockHeight2HashKey(state.id.height)
	hval := make([]byte, 4)
	binary.LittleEndian.PutUint32(hval, state.id.height)
	err = s.db.Update(func(txn *badger.Txn) error {
		// Update account balances.
		for _, acct := range state.changes {
			prev := &big.Int{}
			it := txn.NewIterator(badger.IteratorOptions{
				Reverse: true,
			})
			defer it.Close()
			it.Seek(acct.key)
			var item *badger.Item
			if it.ValidForPrefix(acct.prefix) {
				exists := true
				item = it.Item()
				if bytes.Equal(acct.key, item.Key()) {
					it.Next()
					if it.ValidForPrefix(acct.prefix) {
						item = it.Item()
					} else {
						exists = false
					}
				}
				if exists {
					err := item.Value(func(val []byte) error {
						prev.SetBytes(val)
						return nil
					})
					if err != nil {
						return err
					}
				}
			}
			balance := prev.Add(prev, acct.diff).Bytes()
			if err := txn.Set(acct.key, balance); err != nil {
				return err
			}
		}
		// Write block metadata.
		if err := txn.Set(blockKey, blockData); err != nil {
			return err
		}
		if err := txn.Set(hashKey, hval); err != nil {
			return err
		}
		if err := txn.Set(heightKey, state.id.hash[:]); err != nil {
			return err
		}
		for _, hash := range state.hashes {
			key := make([]byte, 33)
			key[0] = 'e'
			copy(key[1:], hash)
			if err := txn.Set(key, []byte{}); err != nil {
				return err
			}
		}
		return txn.Set([]byte("height"), hval)
	})
	if err != nil {
		return err
	}
	s.setHeight(int64(state.id.height), int64(state.synced))
	return nil
}

func (s *Store) setHeight(indexed int64, synced int64) {
	s.mu.Lock()
	s.heightIndexed = &indexed
	s.heightSynced = &synced
	s.mu.Unlock()
}

func (s *Store) syncStatus() *types.SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	synced := false
	if s.heightSynced != nil {
		synced = *s.heightIndexed == *s.heightSynced
	}
	return &types.SyncStatus{
		CurrentIndex: s.heightIndexed,
		Synced:       &synced,
		TargetIndex:  s.heightSynced,
	}
}

// NOTE(tav): This function must return the exact pointer as a registered
// currency in s.tokens, as it will be used for map lookups.
func (s *Store) validateCurrency(c *types.Currency) (*currencyInfo, *types.Error) {
	if c == nil || c.Metadata == nil {
		return nil, invalidCurrencyf("currency.metadata field missing")
	}
	addr, ok := c.Metadata["contract"]
	if !ok {
		return nil, invalidCurrencyf("currency.metadata.contract field missing")
	}
	raw, ok := addr.(string)
	if !ok {
		return nil, invalidCurrencyf("currency.metadata.contract is not string")
	}
	contract, err := common.AddressFromHexString(raw)
	if err != nil {
		return nil, invalidCurrencyf(
			"unable to parse currency.metadata.contract: %s", err,
		)
	}
	info, ok := s.tokens[contract]
	if !ok {
		return nil, wrapErr(
			errCurrencyNotDefined,
			fmt.Errorf("services: %s is not defined as a currency", raw),
		)
	}
	if info.currency.Decimals != c.Decimals {
		return nil, invalidCurrencyf(
			"mismatching decimals value for currency: expected %d, got %d",
			info.currency.Decimals, c.Decimals,
		)
	}
	if info.currency.Symbol != c.Symbol {
		return nil, invalidCurrencyf(
			"mismatching symbol for currency: expected %q, got %q",
			info.currency.Symbol, c.Symbol,
		)
	}
	return info, nil
}

func NewStore(dir string, oep4 []*OEP4Token, offline bool) (*Store, error) {
	tokens := map[common.Address]*currencyInfo{
		ongAddr: {
			contract: ongAddr,
			currency: &types.Currency{
				Decimals: constants.ONG_DECIMALS_V2,
				Symbol:   constants.ONG_SYMBOL,
				Metadata: map[string]interface{}{
					"contract": ongAddr.ToHexString(),
				},
			},
		},
		ontAddr: {
			contract: ontAddr,
			currency: &types.Currency{
				Decimals: constants.ONT_DECIMALS_V2,
				Symbol:   constants.ONT_SYMBOL,
				Metadata: map[string]interface{}{
					"contract": ontAddr.ToHexString(),
				},
			},
		},
	}
	for _, token := range oep4 {
		tokens[token.Contract] = &currencyInfo{
			contract: token.Contract,
			currency: &types.Currency{
				Decimals: int32(token.Decimals),
				Symbol:   token.Symbol,
				Metadata: map[string]interface{}{
					"contract": token.Contract.ToHexString(),
				},
			},
			wasm: token.Wasm,
		}
	}
	opts := badger.DefaultOptions(dir)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("services: failed to open internal data store: %w", err)
	}
	if offline {
		return &Store{
			db:     db,
			tokens: tokens,
		}, nil
	}
	var indexed *int64
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("height"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			height := int64(binary.LittleEndian.Uint32(val))
			indexed = &height
			return nil
		})
	})
	if err != nil && err != badger.ErrKeyNotFound {
		return nil, fmt.Errorf(
			"services: failed to read height while opening internal data store: %s",
			err,
		)
	}
	synced := int64(actor.GetCurrentBlockHeight())
	parsedAbi, _ := abi.JSON(strings.NewReader(ERC20ABI))
	return &Store{
		db:            db,
		heightIndexed: indexed,
		heightSynced:  &synced,
		tokens:        tokens,
		parsedAbi:     parsedAbi,
	}, nil
}

func accountKeyPrefix(acct []byte, contract []byte) []byte {
	key := append([]byte{'a'}, acct...)
	return append(key, contract...)
}

// Some of the native addresses like ONT/ONG are "compressed", and represented
// by a leading "\x01" byte. Uncompressed addresses are represented by a leading
// null byte.
func addr2slice(addr common.Address) []byte {
	switch addr {
	case ongAddr:
		return []byte{1, 2}
	case ontAddr:
		return []byte{1, 1}
	case govAddr:
		return []byte{1, 7}
	case nullAddr:
		return []byte{1, 0}
	default:
		return append([]byte{0}, addr[:]...)
	}
}

func blockHash2HeightKey(hash []byte) []byte {
	key := make([]byte, 33)
	key[0] = 'c'
	copy(key[1:], hash)
	return key
}

func blockHeight2HashKey(height uint32) []byte {
	key := make([]byte, 5)
	key[0] = 'd'
	binary.LittleEndian.PutUint32(key[1:], height)
	return key
}

func blockKey(height uint32) []byte {
	key := make([]byte, 5)
	key[0] = 'b'
	binary.LittleEndian.PutUint32(key[1:], height)
	return key
}

// Decode state ('transfer', from, to, amount) to a transfer struct.
func decodeTransfer(height uint32, info *event.ExecuteNotify, evt *event.NotifyEventInfo) *transfer {
	elems, ok := evt.States.([]interface{})
	if !ok {
		log.Warnf(
			"Ignoring event for txn %s at height %d: type(state) = %s",
			info.TxHash.ToHexString(), height, reflect.TypeOf(evt.States),
		)
		return nil
	}
	if len(elems) != 4 && len(elems) != 5 {
		log.Warnf(
			"Ignoring event for txn %s at height %d: len(state) != 4 or len(state) != 5",
			info.TxHash.ToHexString(), height,
		)
		return nil
	}
	if evt.ContractAddress == ontAddr || evt.ContractAddress == ongAddr {
		state := make([]string, 4)
		for i := 0; i < 3; i++ {
			rv := reflect.ValueOf(elems[i])
			if rv.Kind() != reflect.String {
				log.Warnf(
					"Ignoring event for txn %s at height %d: type(state[%d]) != string %s",
					info.TxHash.ToHexString(), height, i, rv.Type(),
				)
				return nil
			}
			state[i] = rv.String()
		}
		if state[0] != "transfer" {
			return nil
		}
		from, err := common.AddressFromBase58(state[1])
		if err != nil {
			log.Errorf(
				`Failed to decode "from" for txn %s at height %d: %s`,
				info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		to, err := common.AddressFromBase58(state[2])
		if err != nil {
			log.Errorf(
				`Failed to decode "to" for txn %s at height %d: %s`,
				info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		rv := reflect.ValueOf(elems[3])
		if rv.Type() != jsonNumberType {
			log.Errorf(
				`Unexpected datatype for "amount" for txn %s at height %d: %v`,
				info.TxHash.ToHexString(), height, rv.Type(),
			)
			return nil
		}
		raw := rv.Interface().(json.Number)
		amount, err := raw.Int64()
		if err != nil {
			log.Warnf(
				"Unable to decode transfer amount %q for txn %s at height %d: %s",
				raw, info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		if amount < 0 {
			log.Fatalf(
				"Transfer amount for txn %s at height %d is negative: %d",
				info.TxHash.ToHexString(), height, amount,
			)
		}
		if len(elems) == 5 {
			rv := reflect.ValueOf(elems[4])
			if rv.Type() != jsonNumberType {
				log.Errorf(
					`Unexpected datatype for "value" for txn %s at height %d: %v`,
					info.TxHash.ToHexString(), height, rv.Type(),
				)
				return nil
			}
			raw := rv.Interface().(json.Number)
			value, err := raw.Int64()
			if err != nil {
				log.Warnf(
					"Unable to decode transfer value %q for txn %s at height %d: %s",
					raw, info.TxHash.ToHexString(), height, err,
				)
				return nil
			}
			if value < 0 {
				log.Fatalf(
					"Transfer amount for txn %s at height %d is negative: %d",
					info.TxHash.ToHexString(), height, value,
				)
			}
			res := big.NewInt(0).Mul(big.NewInt(amount), big.NewInt(constants.GWei))
			res.Add(res, big.NewInt(value))
			xfer := &transfer{
				amount: res,
				from:   from,
				to:     to,
			}
			if evt.ContractAddress == ongAddr && to == govAddr && uint64(amount) == info.GasConsumed {
				xfer.isGas = true
			}
			return xfer
		}
		xfer := &transfer{
			amount: big.NewInt(amount),
			from:   from,
			to:     to,
		}
		if evt.ContractAddress == ongAddr && to == govAddr && uint64(amount) == info.GasConsumed {
			xfer.isGas = true
		}
		return xfer
	}
	state := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		elem := reflect.ValueOf(elems[i])
		if elem.Kind() != reflect.String {
			log.Errorf(
				"Ignoring event for txn %s at height %d: type(state[%d]) != string",
				info.TxHash.ToHexString(), height, i,
			)
			return nil
		}
		val, err := hex.DecodeString(elem.String())
		if err != nil {
			log.Errorf(
				"Failed to decode state[%d] for txn %s at height %d: %s",
				i, info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		state[i] = val
	}
	if !bytes.EqualFold(state[0], []byte("transfer")) {
		return nil
	}
	xfer := &transfer{}
	if !isNull(state[1]) {
		from, err := common.AddressParseFromBytes(state[1])
		if err != nil {
			log.Errorf(
				`Failed to decode "from" for txn %s at height %d: %s`,
				info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		xfer.from = from
	}
	if !isNull(state[2]) {
		to, err := common.AddressParseFromBytes(state[2])
		if err != nil {
			log.Errorf(
				`Failed to decode "to" for txn %s at height %d: %s`,
				info.TxHash.ToHexString(), height, err,
			)
			return nil
		}
		xfer.to = to
	}
	amount := common.BigIntFromNeoBytes(state[3])
	if amount.Cmp(big.NewInt(0)) == -1 {
		log.Errorf(
			"Transfer amount for txn %s at height %d outside of expected range: %v",
			info.TxHash.ToHexString(), height, amount,
		)
		return nil
	}
	xfer.amount = amount
	/*
		if amount.Uint64()%constants.GWei != 0 {
			elem := reflect.ValueOf(elems[4])
			if elem.Kind() != reflect.String {
				log.Errorf(
					"Ignoring event for txn %s at height %d: type(state[4]) != string",
					info.TxHash.ToHexString(), height,
				)
				return nil
			}
			val, err := hex.DecodeString(elem.String())
			if err != nil {
				log.Errorf(
					"Failed to decode state[5] for txn %s at height %d: %s",
					info.TxHash.ToHexString(), height, err,
				)
				return nil
			}
			value := common.BigIntFromNeoBytes(val)
			if value.Cmp(big.NewInt(0)) == -1 {
				log.Errorf(
					"Transfer amount for txn %s at height %d outside of expected range: %v",
					info.TxHash.ToHexString(), height, value,
				)
				return nil
			}
			totalAmount := &big.Int{}
			xfer.amount = totalAmount.Add(amount, value)
		} else {
			xfer.amount = amount
		}
	*/
	return xfer
}

func checkEvmEventLog(evt *event.NotifyEventInfo) (bool, *ctypes.StorageLog) {
	ethLog, err := event.NotifyEventInfoToEvmLog(evt)
	if err != nil {
		return false, nil
	}
	return true, ethLog
}
func parseEvmOngTransferLog(ethLog *ctypes.StorageLog, parsedAbi abi.ABI, gasConsumed uint64) (*transfer, error) {
	ongLog := types2.Log{
		Address: ethLog.Address,
		Topics:  ethLog.Topics,
		Data:    ethLog.Data,
	}
	if ongLog.Address == ONG_ADDR {
		nbc := bind.NewBoundContract(ethcom.Address{}, parsedAbi, nil, nil, nil)
		tf := new(ERC20Transfer)
		err := nbc.UnpackLog(tf, "Transfer", ongLog)
		if err != nil {
			return nil, err
		}
		xfer := &transfer{
			amount: tf.Value,
			from:   common.Address(tf.From),
			to:     common.Address(tf.To),
		}
		if tf.To == GOV_ADDR && tf.Value.Uint64()/constants.GWei == gasConsumed {
			xfer.isGas = true
		}
		return xfer, nil
	} else {
		return nil, fmt.Errorf("parse evm ong transfer error")
	}
}

func balanceCal(xfer *transfer, evt *event.NotifyEventInfo, diffs map[common.Address]map[common.Address]*big.Int) *model.Transfer {
	if xfer.from != nullAddr {
		accts, ok := diffs[xfer.from]
		if ok {
			balance, ok := accts[evt.ContractAddress]
			if ok {
				accts[evt.ContractAddress] = balance.Sub(balance, xfer.amount)
			} else {
				accts[evt.ContractAddress] = (&big.Int{}).Neg(xfer.amount)
			}
		} else {
			diffs[xfer.from] = map[common.Address]*big.Int{
				evt.ContractAddress: (&big.Int{}).Neg(xfer.amount),
			}
		}
	}
	if xfer.to != nullAddr {
		accts, ok := diffs[xfer.to]
		if ok {
			balance, ok := accts[evt.ContractAddress]
			if ok {
				accts[evt.ContractAddress] = balance.Add(balance, xfer.amount)
			} else {
				accts[evt.ContractAddress] = xfer.amount
			}
		} else {
			diffs[xfer.to] = map[common.Address]*big.Int{
				evt.ContractAddress: xfer.amount,
			}
		}
	}
	return &model.Transfer{
		Amount:   xfer.amount.Bytes(),
		Contract: addr2slice(evt.ContractAddress),
		From:     addr2slice(xfer.from),
		IsGas:    xfer.isGas,
		To:       addr2slice(xfer.to),
	}
}

//return gasVerified,isGas,isContinue
func checkgasVerified(contractAddress, payer, from common.Address, failed, gasVerified, isGas bool) (bool, bool, bool) {
	if failed {
		if contractAddress != ongAddr {
			return false, false, true
		}
		// NOTE(tav): We skip additional transfer events on
		// failed transactions if we've already matched the gas
		// fee.
		if gasVerified {
			return false, false, true
		}
	}
	if gasVerified {
		return gasVerified, false, false
	} else if isGas {
		if payer == from {
			return true, isGas, false
		} else {
			return gasVerified, false, false
		}
	}
	return false, false, false
}
func decompressAddr(xs []byte) common.Address {
	switch len(xs) {
	case 2:
		switch xs[1] {
		case 2:
			return ongAddr
		case 1:
			return ontAddr
		case 7:
			return govAddr
		case 0:
			return nullAddr
		default:
			panic(fmt.Errorf("invalid address to decompress: %q", string(xs)))
		}
	default:
		addr, err := common.AddressParseFromBytes(xs[1:])
		if err != nil {
			panic(err)
		}
		return addr
	}
}

func isNull(v []byte) bool {
	return len(v) == 1 && v[0] == 0
}

func slice2addr(xs []byte) (common.Address, error) {
	switch len(xs) {
	case 2:
		if xs[0] != 1 {
			return common.ADDRESS_EMPTY, fmt.Errorf(
				"services: unexpected slice address value: %x", xs,
			)
		}
		switch xs[1] {
		case 2:
			return ongAddr, nil
		case 1:
			return ontAddr, nil
		case 7:
			return govAddr, nil
		case 0:
			return nullAddr, nil
		default:
			return common.ADDRESS_EMPTY, fmt.Errorf(
				"services: unexpected slice address value: %x", xs,
			)
		}
	case 0:
		return common.ADDRESS_EMPTY, fmt.Errorf(
			"services: empty slice address value",
		)
	default:
		addr, err := common.AddressParseFromBytes(xs[1:])
		if err != nil {
			return common.ADDRESS_EMPTY, fmt.Errorf(
				"services: failed to decode slice address value %x: %s",
				xs, err,
			)
		}
		return addr, nil
	}
}

func init() {
	// Enable the use of json.Number when decoding event state from the ledger.
	ledgerstore.UseNumber = true
}
