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
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"reflect"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-rosetta/chain"
	"github.com/ontio/ontology-rosetta/log"
	"github.com/ontio/ontology-rosetta/model"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/signature"
	ctypes "github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/core/utils"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/http/base/actor"
	"google.golang.org/protobuf/proto"
)

// TODO(tav): Figure out whether to include the sub account in account
// identifiers.

// ConstructionCombine implements the /construction/combine endpoint.
func (s *service) ConstructionCombine(ctx context.Context, r *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	txn, xerr := decodeTransaction(r.UnsignedTransaction)
	if xerr != nil {
		return nil, xerr
	}
	mut, err := txn.IntoMutable()
	if err != nil {
		return nil, wrapErr(errInvalidTransactionPayload, err)
	}
	if len(mut.Sigs) > 0 {
		return nil, wrapErr(
			errInvalidTransactionPayload,
			fmt.Errorf("services: unexpected signature found in unsigned transaction"),
		)
	}
	sigs := r.Signatures
	if len(sigs) == 0 {
		return nil, errInvalidSignature
	}
	// TODO(ZhouPW): How should we handle the multi-sig address case?
	for _, sig := range sigs {
		if sig.PublicKey == nil {
			return nil, errInvalidPublicKey
		}
		if len(sig.PublicKey.Bytes) < 1 {
			return nil, errInvalidPublicKey
		}
		// NOTE(tav): We don't currently validate the key/signature types.
		pk, err := keypair.DeserializePublicKey(sig.PublicKey.Bytes)
		if err != nil {
			return nil, errInvalidPublicKey
		}
		if sig.SigningPayload == nil {
			return nil, errInvalidSignature
		}
		err = signature.Verify(pk, sig.SigningPayload.Bytes, sig.Bytes)
		if err != nil {
			return nil, wrapErr(errInvalidSignature, err)
		}
		mut.Sigs = append(mut.Sigs, ctypes.Sig{
			M:       1,
			PubKeys: []keypair.PublicKey{pk},
			SigData: [][]byte{sig.Bytes},
		})
	}
	txn, err = mut.IntoImmutable()
	if err != nil {
		return nil, wrapErr(errInternal, err)
	}
	sink := common.ZeroCopySink{}
	txn.Serialization(&sink)
	return &types.ConstructionCombineResponse{
		SignedTransaction: hex.EncodeToString(sink.Bytes()),
	}, nil
}

// ConstructionDerive implements the /construction/derive endpoint.
func (s *service) ConstructionDerive(ctx context.Context, r *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error) {
	if r.PublicKey == nil {
		return nil, errInvalidPublicKey
	}
	pk, err := keypair.DeserializePublicKey(r.PublicKey.Bytes)
	if err != nil {
		return nil, errInvalidPublicKey
	}
	addr := ctypes.AddressFromPubKey(pk)
	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr.ToBase58(),
		},
	}, nil
}

// ConstructionHash implements the /construction/hash endpoint.
func (s *service) ConstructionHash(ctx context.Context, r *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	txn, xerr := decodeTransaction(r.SignedTransaction)
	if xerr != nil {
		return nil, xerr
	}
	return txhash2response(txn.Hash())
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *service) ConstructionMetadata(ctx context.Context, r *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	opts := &model.ConstructOptions{}
	if xerr := decodeProtobuf(r.Options, opts); xerr != nil {
		return nil, xerr
	}
	gasPrice, err := getGasPrice()
	// NOTE(tav): We assume that the gas prices will only go up over time, which
	// may not be true.
	if err == nil && gasPrice > opts.GasPrice {
		opts.GasPrice = gasPrice
	} else if opts.GasPrice < defaultGasPrice {
		opts.GasPrice = defaultGasPrice
	}
	if opts.GasLimit < minGasLimit {
		opts.GasLimit = minGasLimit
	}
	if opts.Nonce == 0 {
		buf := make([]byte, 8)
		for i := 0; i < 100; i++ {
			n, err := rand.Read(buf)
			if err != nil || n != 8 {
				return nil, errNonceGenerationFailed
			}
			opts.Nonce = binary.LittleEndian.Uint32(buf)
			txn, err := s.constructTransfer(opts)
			if err != nil {
				return nil, wrapErr(errInvalidConstructOptions, err)
			}
			exists, xerr := s.store.checkTxHash(txn.Hash())
			if xerr != nil {
				return nil, xerr
			}
			if !exists {
				break
			}
		}
		if opts.Nonce == 0 {
			return nil, errNonceGenerationFailed
		}
	} else {
		txn, err := s.constructTransfer(opts)
		if err != nil {
			return nil, wrapErr(errInvalidConstructOptions, err)
		}
		exists, xerr := s.store.checkTxHash(txn.Hash())
		if xerr != nil {
			return nil, xerr
		}
		if exists {
			return nil, wrapErr(
				errInvalidNonce,
				fmt.Errorf(
					"a transaction already exists with the same hash for nonce %d",
					opts.Nonce,
				),
			)
		}
	}
	enc, err := proto.Marshal(opts)
	if err != nil {
		return nil, wrapErr(errProtobuf, err)
	}
	return &types.ConstructionMetadataResponse{
		Metadata: map[string]interface{}{
			"protobuf": hex.EncodeToString(enc),
		},
	}, nil
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *service) ConstructionParse(ctx context.Context, r *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error) {
	txn, xerr := decodeTransaction(r.Transaction)
	if xerr != nil {
		return nil, xerr
	}
	ops, xerr := s.parsePayload(txn.Payload)
	if xerr != nil {
		return nil, xerr
	}
	// NOTE(tav): We assume that we're dealing with a transaction created by
	// ourselves, and that parsePayload will only return 2 operations. The first
	// for the "transfer from", and the second for the "transfer to".
	if len(ops) != 2 {
		return nil, wrapErr(
			errInternal,
			fmt.Errorf("unexpected number of operations in transaction: %d", len(ops)),
		)
	}
	if ops[0].Amount == nil || len(ops[0].Amount.Value) == 0 || ops[0].Amount.Value[0] != '-' {
		return nil, wrapErr(
			errInternal,
			fmt.Errorf(`unexpected "transfer from" operation in transaction: %v`, ops[0]),
		)
	}
	var signers []*types.AccountIdentifier
	if r.Signed {
		acct := ops[0].Account
		signers = append(signers, acct)
		if acct.Address != txn.Payer.ToBase58() {
			payer := *acct
			payer.Address = txn.Payer.ToBase58()
			signers = append(signers, &payer)
		}
	}
	enc, err := proto.Marshal(&model.ConstructOptions{
		GasLimit: txn.GasLimit,
		GasPrice: txn.GasPrice,
		Payer:    txn.Payer[:],
	})
	if err != nil {
		return nil, wrapErr(errProtobuf, err)
	}
	return &types.ConstructionParseResponse{
		AccountIdentifierSigners: signers,
		Metadata: map[string]interface{}{
			"protobuf": enc,
		},
		Operations: ops,
	}, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *service) ConstructionPayloads(ctx context.Context, r *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	opts := &model.ConstructOptions{}
	if xerr := decodeProtobuf(r.Metadata, opts); xerr != nil {
		return nil, xerr
	}
	xfer, xerr := s.validateOps(r.Operations)
	if xerr != nil {
		return nil, xerr
	}
	if !bytes.Equal(opts.Amount, xfer.amount.Bytes()) {
		return nil, invalidConstructf("amount does not match value from operations")
	}
	if !bytes.Equal(opts.Contract, xfer.contract[:]) {
		return nil, invalidConstructf("amount does not match value from operations")
	}
	if !bytes.Equal(opts.From, xfer.from[:]) {
		return nil, invalidConstructf("amount does not match value from operations")
	}
	if !bytes.Equal(opts.To, xfer.to[:]) {
		return nil, invalidConstructf("amount does not match value from operations")
	}
	txn, err := s.constructTransfer(opts)
	if err != nil {
		return nil, wrapErr(errInvalidConstructOptions, err)
	}
	sink := common.ZeroCopySink{}
	txn.Serialization(&sink)
	hash := txn.Hash()
	acct := &types.AccountIdentifier{
		Address: xfer.from.ToBase58(),
	}
	if !xfer.isNative() {
		acct.SubAccount = &types.SubAccountIdentifier{
			Address: xfer.contract.ToHexString(),
		}
	}
	payloads := []*types.SigningPayload{{
		AccountIdentifier: acct,
		Bytes:             hash[:],
	}}
	if txn.Payer != xfer.from {
		payer := *acct
		payer.Address = txn.Payer.ToBase58()
		payloads = append(payloads, &types.SigningPayload{
			AccountIdentifier: &payer,
			Bytes:             hash[:],
		})
	}
	return &types.ConstructionPayloadsResponse{
		Payloads:            payloads,
		UnsignedTransaction: hex.EncodeToString(sink.Bytes()),
	}, nil
}

// ConstructionPreprocess implements the /construction/preprocess endpoint.
func (s *service) ConstructionPreprocess(ctx context.Context, r *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	if len(r.MaxFee) > 0 {
		return nil, wrapErr(
			errInvalidRequestField,
			fmt.Errorf("services: unsupported field: max_fee"),
		)
	}
	if r.SuggestedFeeMultiplier != nil {
		return nil, wrapErr(
			errInvalidRequestField,
			fmt.Errorf("services: unsupported field: suggested_fee_multiplier"),
		)
	}
	gasLimit, err := getUint64Field(r.Metadata, "gas_limit")
	if err != nil {
		return nil, wrapErr(errInvalidGasLimit, err)
	}
	gasPrice, err := getUint64Field(r.Metadata, "gas_price")
	if err != nil {
		return nil, wrapErr(errInvalidGasPrice, err)
	}
	nonce, err := getUint64Field(r.Metadata, "nonce")
	if err != nil {
		return nil, wrapErr(errInvalidNonce, err)
	}
	payer, xerr := getPayer(r.Metadata)
	if xerr != nil {
		return nil, xerr
	}
	xfer, xerr := s.validateOps(r.Operations)
	if xerr != nil {
		return nil, xerr
	}
	if payer != common.ADDRESS_EMPTY {
		payer = xfer.from
	}
	enc, err := proto.Marshal(&model.ConstructOptions{
		Amount:   xfer.amount.Bytes(),
		Contract: xfer.contract[:],
		From:     xfer.from[:],
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Nonce:    uint32(nonce),
		Payer:    payer[:],
		To:       xfer.to[:],
	})
	if err != nil {
		return nil, wrapErr(errProtobuf, err)
	}
	return &types.ConstructionPreprocessResponse{
		Options: map[string]interface{}{
			"protobuf": hex.EncodeToString(enc),
		},
	}, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *service) ConstructionSubmit(ctx context.Context, r *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.offline {
		return nil, errOfflineMode
	}
	txn, xerr := decodeTransaction(r.SignedTransaction)
	if xerr != nil {
		return nil, xerr
	}
	if err, desc := actor.AppendTxToPool(txn); err != errors.ErrNoError {
		log.Errorf("Failed to broadcast transaction: %s (%s)", err, desc)
		return nil, wrapErr(errBroadcastFailed, err)
	}
	return txhash2response(txn.Hash())
}

func (s *service) constructTransfer(opts *model.ConstructOptions) (*ctypes.Transaction, error) {
	contract, err := common.AddressParseFromBytes(opts.Contract)
	if err != nil {
		return nil, err
	}
	from, err := common.AddressParseFromBytes(opts.From)
	if err != nil {
		return nil, err
	}
	payer, err := common.AddressParseFromBytes(opts.Payer)
	if err != nil {
		return nil, err
	}
	to, err := common.AddressParseFromBytes(opts.To)
	if err != nil {
		return nil, err
	}
	// TODO(tav): Check whether the params need to be nested, e.g. for native vs
	// OEP4 contracts.
	code, err := utils.BuildNativeInvokeCode(
		contract,
		0,
		"transfer",
		[]interface{}{from, to, (&big.Int{}).SetBytes(opts.Amount)},
	)
	if err != nil {
		return nil, fmt.Errorf("services: unable to build transaction invoke code: %s", err)
	}
	// TODO(tav): Special case WASM contracts?
	mut := &ctypes.MutableTransaction{
		GasLimit: opts.GasLimit,
		GasPrice: opts.GasPrice,
		Nonce:    opts.Nonce,
		Payer:    payer,
		Payload: &payload.InvokeCode{
			Code: code,
		},
		Sigs:   []ctypes.Sig{},
		TxType: ctypes.InvokeNeo,
	}
	return mut.IntoImmutable()
}

func (s *service) parsePayload(p ctypes.Payload) ([]*types.Operation, *types.Error) {
	if p == nil {
		return nil, errInvalidTransactionPayload
	}
	invoke, ok := p.(*payload.InvokeCode)
	if !ok || invoke == nil {
		return nil, errInvalidTransactionPayload
	}
	xfers, contract, err := chain.ParsePayload(invoke.Code)
	if err != nil {
		return nil, wrapErr(errInvalidTransactionPayload, err)
	}
	info, xerr := s.store.getCurrencyInfo(contract)
	if xerr != nil {
		return nil, xerr
	}
	ops := []*types.Operation{}
	for _, xfer := range xfers {
		ops = s.appendOperations(ops, &transferInfo{
			amount:   xfer.Amount,
			contract: contract,
			currency: info.currency,
			from:     xfer.From,
			to:       xfer.To,
		})
	}
	return ops, nil
}

// NOTE(tav): We currently only support a simple transfer of an asset from one
// account to another.
func (s *service) validateOps(ops []*types.Operation) (*transferInfo, *types.Error) {
	if ops == nil {
		return nil, invalidOpsf("missing operations field")
	}
	if len(ops) != 2 {
		return nil, invalidOpsf("unexpected number of operations: %d", len(ops))
	}
	addrs := make([]common.Address, 2)
	amounts := make([]*big.Int, 2)
	zero := big.NewInt(0)
	var cinfo *currencyInfo
	for i, op := range ops {
		if op.Account == nil {
			return nil, invalidOpsf("missing operations[%d].account", i)
		}
		addr, err := common.AddressFromBase58(op.Account.Address)
		if err != nil {
			return nil, invalidOpsf(
				"unable to parse operations[%d].account.address: %s",
				i, err,
			)
		}
		addrs[i] = addr
		if op.Amount == nil {
			return nil, invalidOpsf("missing operations[%d].amount", i)
		}
		amount, ok := (&big.Int{}).SetString(op.Amount.Value, 10)
		if !ok {
			return nil, invalidOpsf(
				"invalid operations[%d].amount.value: %s",
				i, op.Amount.Value,
			)
		}
		if amount.Cmp(zero) == 0 {
			return nil, invalidOpsf("operations[%d].amount.value is zero", i)
		}
		amounts[i] = amount
		token, xerr := s.store.validateCurrency(op.Amount.Currency)
		if xerr != nil {
			return nil, xerr
		}
		if token.isNative() {
			if op.Account.SubAccount == nil {
				return nil, invalidOpsf("missing operations[%d].account.sub_account", i)
			}
			caddr, err := common.AddressFromHexString(op.Account.SubAccount.Address)
			if err != nil {
				return nil, invalidOpsf(
					"unable to parse operations[%d].account.sub_account.address: %s",
					i, err,
				)
			}
			if token.contract != caddr {
				return nil, invalidOpsf(
					"operations[%d].account.sub_account.address does not match currency",
					i,
				)
			}
		}
		if cinfo == nil {
			cinfo = token
		} else if cinfo != token {
			return nil, invalidOpsf("operations must be in the same currency")
		}
		if op.OperationIdentifier == nil {
			return nil, invalidOpsf("missing operations[%d].operation_identifier", i)
		}
		if op.Type != opTransfer {
			return nil, invalidOpsf("unsupported operation type: %q", op.Type)
		}
	}
	switch {
	case len(ops[0].RelatedOperations) > 0:
		xerr := validateRelation(ops, 0, 1)
		if xerr != nil {
			return nil, xerr
		}
	case len(ops[1].RelatedOperations) > 0:
		xerr := validateRelation(ops, 1, 0)
		if xerr != nil {
			return nil, xerr
		}
	default:
		return nil, invalidOpsf("invalid related_operations on operations")
	}
	sum := (&big.Int{}).Add(amounts[0], amounts[1])
	if sum.Cmp(zero) != 0 {
		return nil, invalidOpsf("amount values in operations do not sum to zero")
	}
	xfer := &transferInfo{
		contract: cinfo.contract,
		currency: cinfo.currency,
	}
	switch amounts[0].Cmp(zero) {
	case 1:
		xfer.amount = amounts[0]
		xfer.from = addrs[1]
		xfer.to = addrs[0]
	case -1:
		xfer.amount = amounts[1]
		xfer.from = addrs[0]
		xfer.to = addrs[1]
	default:
		return nil, invalidOpsf("amount values in operations cannot be zero")
	}
	if xfer.from == common.ADDRESS_EMPTY {
		return nil, invalidOpsf("transfers from null addresses are not supported")
	}
	return xfer, nil
}

func decodeProtobuf(md map[string]interface{}, m proto.Message) *types.Error {
	data, ok := md["protobuf"]
	if !ok {
		return wrapErr(errProtobuf, fmt.Errorf("services: protobuf metadata field is missing"))
	}
	raw, ok := data.(string)
	if !ok {
		return wrapErr(errProtobuf, fmt.Errorf("services: protobuf metadata field is not a string"))
	}
	val, err := hex.DecodeString(raw)
	if err != nil {
		return wrapErr(errProtobuf, err)
	}
	if err := proto.Unmarshal(val, m); err != nil {
		return wrapErr(errProtobuf, err)
	}
	return nil
}

func decodeTransaction(data string) (*ctypes.Transaction, *types.Error) {
	if len(data) == 0 {
		return nil, errInvalidTransactionPayload
	}
	raw, err := hex.DecodeString(data)
	if err != nil {
		return nil, wrapErr(errInvalidTransactionPayload, err)
	}
	txn, err := ctypes.TransactionFromRawBytes(raw)
	if err != nil {
		return nil, wrapErr(errInvalidTransactionPayload, err)
	}
	if txn == nil {
		return nil, wrapErr(
			errInvalidTransactionPayload,
			fmt.Errorf("transaction is nil when decoded"),
		)
	}
	return txn, nil
}

func getGasPrice() (uint64, error) {
	var end uint32 = 0
	var price uint64 = 0
	start := actor.GetCurrentBlockHeight()
	if start > 100 {
		end = start - 100
	}
	for height := start; height >= end; height-- {
		hdr, err := actor.GetHeaderByHeight(height)
		if err == nil && hdr.TransactionsRoot != common.UINT256_EMPTY {
			block, err := actor.GetBlockByHeight(height)
			if err != nil {
				log.Errorf("Failed to get block at height %d: %s", height, err)
				return 0, err
			}
			for _, txn := range block.Transactions {
				price += txn.GasPrice
			}
			price = price / uint64(len(block.Transactions))
			break
		}
	}
	return price, nil
}

func getPayer(md map[string]interface{}) (common.Address, *types.Error) {
	if md == nil {
		return common.ADDRESS_EMPTY, nil
	}
	val, ok := md["payer"]
	if !ok {
		return common.ADDRESS_EMPTY, nil
	}
	raw, ok := val.(string)
	if !ok {
		return common.ADDRESS_EMPTY, wrapErr(
			errInvalidPayerAddress,
			fmt.Errorf(
				"services: unexpected datatype for metadata.payer: %s",
				reflect.TypeOf(val),
			),
		)
	}
	addr, err := common.AddressFromBase58(raw)
	if err != nil {
		return common.ADDRESS_EMPTY, wrapErr(
			errInvalidPayerAddress,
			fmt.Errorf("services: unable to parse metadata.payer: %s", err),
		)
	}
	return addr, nil
}

func getUint64Field(md map[string]interface{}, field string) (uint64, error) {
	if md == nil {
		return 0, nil
	}
	val, ok := md[field]
	if !ok {
		return 0, nil
	}
	raw, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("services: unexpected datatype for metadata.%s: %s", field, val)
	}
	v := uint64(raw)
	if float64(v) != raw {
		return 0, fmt.Errorf(
			"services: cannot accurately cast metadata.%s value to uint64: %v",
			field, raw,
		)
	}
	switch field {
	case "gas_limit":
		if v == 0 {
			v = minGasLimit
		}
		if v < minGasLimit {
			return 0, fmt.Errorf(
				"services: gas limit of %d is below the minimum value of %d",
				v, minGasLimit,
			)
		}
		return v, nil
	case "gas_price":
		return defaultGasPrice, nil
	case "nonce":
		if v > math.MaxUint32 {
			return 0, fmt.Errorf("services: nonce value %d is outside the uint32 range", v)
		}
		return v, nil
	}
	return v, nil
}

func txhash2response(hash common.Uint256) (*types.TransactionIdentifierResponse, *types.Error) {
	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash.ToHexString(),
		},
	}, nil
}

func validateRelation(ops []*types.Operation, ifrom int, ito int) *types.Error {
	if len(ops[ito].RelatedOperations) > 0 {
		return invalidOpsf(
			"cannot have related_operations on both operations[%d] and operations[%d]",
			ifrom, ito,
		)
	}
	rel := ops[ifrom].RelatedOperations[0]
	if rel == nil {
		return invalidOpsf("invalid operations[%d].related_operations", ifrom)
	}
	src := ops[ito].OperationIdentifier.Index
	if rel.Index != src {
		return invalidOpsf(
			"operations[%d].related_operations does not match operations[%d].operation_identifier",
			ifrom, ito,
		)
	}
	diff := ops[ifrom].OperationIdentifier.Index - src
	if diff != 1 {
		return invalidOpsf(
			"operations[%d].related_operations does not follow from operations[%d]",
			ifrom, ito,
		)
	}
	return nil
}
