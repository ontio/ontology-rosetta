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
	"strconv"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology-crypto/keypair"
	log "github.com/ontio/ontology-rosetta/common"
	"github.com/ontio/ontology-rosetta/config"
	db "github.com/ontio/ontology-rosetta/store"
	util "github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/signature"
	ctypes "github.com/ontio/ontology/core/types"
	ontErrors "github.com/ontio/ontology/errors"
	bcomn "github.com/ontio/ontology/http/base/common"
)

type ConstructionAPIService struct {
	network *types.NetworkIdentifier
	store   *db.Store
}

func (c ConstructionAPIService) ConstructionCombine(
	ctx context.Context,
	req *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {

	utbytes, err := hex.DecodeString(req.UnsignedTransaction)
	if err != nil {
		return nil, TRANSACTION_HEX_ERROR
	}
	signs := req.Signatures
	if len(signs) == 0 {
		return nil, NO_SIGS_ERROR
	}

	//todo how to solve multi-sign addr case
	ontsigns := make([]ctypes.Sig, len(signs))
	for i, s := range signs {
		pk, err := util.TransformPubkey(s.PublicKey)
		if err != nil {
			return nil, PUBKEY_HEX_ERROR
		}
		sigdata := s.Bytes
		err = signature.Verify(pk, s.SigningPayload.Bytes, sigdata)
		if err != nil {
			return nil, INVALID_SIG_ERROR
		}

		sig := ctypes.Sig{
			SigData: [][]byte{sigdata},
			PubKeys: []keypair.PublicKey{pk},
			M:       1,
		}
		ontsigns[i] = sig
	}

	tx, err := ctypes.TransactionFromRawBytes(utbytes)
	if err != nil {
		return nil, TRANSACTION_HEX_ERROR
	}
	mt, err := tx.IntoMutable()
	if err != nil {
		return nil, TRANSACTION_HEX_ERROR
	}
	mt.Sigs = ontsigns

	imtx, err := mt.IntoImmutable()
	if err != nil {
		return nil, TRANSACTION_HEX_ERROR
	}
	resp := new(types.ConstructionCombineResponse)

	sink := common.ZeroCopySink{}
	imtx.Serialization(&sink)
	resp.SignedTransaction = hex.EncodeToString(sink.Bytes())

	return resp, nil
}

func (c ConstructionAPIService) ConstructionDerive(
	ctx context.Context,
	req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {

	pubkey := req.PublicKey
	//ct := pubkey.CurveType
	meta := req.Metadata
	bts := pubkey.Bytes
	pk, err := keypair.DeserializePublicKey(bts)
	if err != nil {
		return nil, PUBKEY_HEX_ERROR
	}
	addr := ctypes.AddressFromPubKey(pk)

	resp := new(types.ConstructionDeriveResponse)
	// currently we only support base58 or hex format
	// currently we only support base58 or hex format
	if meta == nil {
		resp.Address = addr.ToBase58()
	} else if meta["type"] == strings.ToLower("hex") {
		resp.Address = addr.ToHexString()
	} else if meta["type"] == strings.ToLower("base58") {
		resp.Address = addr.ToBase58()
	} else {
		return nil, INVALID_ADDRESS_TYPE_ERROR
	}
	resp.Metadata = meta

	return resp, nil
}

func (c ConstructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.ConstructionHashResponse, *types.Error) {
	resp := &types.ConstructionHashResponse{}
	bys, err := common.HexToBytes(request.SignedTransaction)
	if err != nil {
		return resp, PARAMS_ERROR
	}
	txn, err := ctypes.TransactionFromRawBytes(bys)
	if err != nil {
		return resp, PARAMS_ERROR
	}
	var hash = txn.Hash()
	resp.TransactionHash = hash.ToHexString()
	return resp, nil
}

func (c ConstructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	resp := &types.ConstructionParseResponse{
		Signers:    make([]string, 0),
		Operations: []*types.Operation{},
		Metadata:   make(map[string]interface{}),
	}
	txData, err := hex.DecodeString(request.Transaction)
	if err != nil {
		return resp, PARAMS_ERROR
	}
	tx, err := ctypes.TransactionFromRawBytes(txData)
	if err != nil {
		return resp, PARAMS_ERROR
	}
	if tx == nil {
		return resp, PARAMS_ERROR
	}
	invokeCode, ok := tx.Payload.(*payload.InvokeCode)
	if !ok {
		log.RosettaLog.Errorf("ConstructionParse: invalid tx payload")
		return resp, INVALID_PAYLOAD
	}
	resp.Metadata[util.PAYER] = tx.Payer.ToBase58()
	transferState, contract, err := util.ParsePayload(invokeCode.Code)
	if err != nil {
		log.RosettaLog.Errorf("ConstructionParse: %s", err)
		return resp, INVALID_PAYLOAD
	}
	currency, ok := util.Currencies[strings.ToLower(contract.ToHexString())]
	if !ok {
		log.RosettaLog.Errorf("ConstructionParse: tx currency %s not exist", contract.ToHexString())
		return resp, CURRENCY_NOT_CONFIG
	}
	for i, state := range transferState {
		operationFrom := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: 2 * int64(i)},
			Type:                config.OP_TYPE_TRANSFER,
			Status:              config.STATUS_SUCCESS.Status,
			Account: &types.AccountIdentifier{
				Address: state.From.ToBase58(),
			},
			Amount: &types.Amount{
				Value:    fmt.Sprintf("-%d", state.Value),
				Currency: currency,
			},
		}
		if request.Signed {
			resp.Signers = append(resp.Signers, state.From.ToBase58())
		}
		resp.Operations = append(resp.Operations, operationFrom)
		operationTo := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: 2*int64(i) + 1},
			RelatedOperations: []*types.OperationIdentifier{
				{Index: operationFrom.OperationIdentifier.Index},
			},
			Type:   config.OP_TYPE_TRANSFER,
			Status: config.STATUS_SUCCESS.Status,
			Account: &types.AccountIdentifier{
				Address: state.To.ToBase58(),
			},
			Amount: &types.Amount{
				Value:    fmt.Sprint(state.Value),
				Currency: currency,
			},
			Metadata: make(map[string]interface{}),
		}
		operationTo.Metadata[util.GAS_PRICE] = tx.GasLimit
		operationTo.Metadata[util.GAS_LIMIT] = tx.GasPrice
		resp.Operations = append(resp.Operations, operationTo)
	}
	return resp, nil
}

func (c ConstructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	resp := &types.ConstructionPayloadsResponse{
		Payloads: make([]*types.SigningPayload, 0),
	}
	payerAddr := request.Metadata[util.PAYER].(string)
	var gasPrice, gasLimit float64
	var fromAddr, toAddr, fromAmount, toAmount, fromSymbol, toSymbol string
	var fromDecimals, toDecimals int32
	for _, operation := range request.Operations {
		if operation.OperationIdentifier.Index == 0 {
			fromAddr = operation.Account.Address
			fromAmount = operation.Amount.Value
			fromSymbol = operation.Amount.Currency.Symbol
			fromDecimals = operation.Amount.Currency.Decimals
		}
		if operation.OperationIdentifier.Index == 1 {
			for _, relatedOperation := range operation.RelatedOperations {
				if relatedOperation.Index == 0 {
					continue
				}
			}
			gasprice := operation.Metadata[util.GAS_PRICE]
			var ok bool
			gasPrice, ok = gasprice.(float64)
			if !ok {
				return resp, PARSE_GAS_PRICE_ERORR
			}
			gaslimit := operation.Metadata[util.GAS_LIMIT]
			gasLimit, ok = gaslimit.(float64)
			if !ok {
				return resp, PARSE_LIMIT_PRICE_ERORR
			}
			toAddr = operation.Account.Address
			toAmount = operation.Amount.Value
			toSymbol = operation.Amount.Currency.Symbol
			toDecimals = operation.Amount.Currency.Decimals
		}
	}
	if fromSymbol != toSymbol || fromDecimals != toDecimals || fromAmount[1:] != toAmount {
		return resp, PARAMS_ERROR
	}
	amount, err := strconv.ParseUint(toAmount, 10, 64)
	if err != nil {
		return resp, PARAMS_ERROR
	}
	mutTx, err := utils.TransferTx(uint64(gasPrice), uint64(gasLimit), toSymbol, fromAddr, toAddr, amount)
	if err != nil {
		return resp, TRANSFER_TX_ERROR
	}
	if payerAddr != "" {
		payer, err := common.AddressFromBase58(payerAddr)
		if err != nil {
			return resp, PAYER_ERROR
		}
		mutTx.Payer = payer
	}
	tx, err := mutTx.IntoImmutable()
	if err != nil {
		return resp, TX_INTO_IMMUTABLE_ERROR
	}
	sink := common.ZeroCopySink{}
	tx.Serialization(&sink)
	payLoad := &types.SigningPayload{
		Address:       fromAddr,
		Bytes:         sink.Bytes(),
		SignatureType: types.Ecdsa,
	}
	resp.UnsignedTransaction = hex.EncodeToString(sink.Bytes())
	resp.Payloads = append(resp.Payloads, payLoad)
	return resp, nil
}

func (c ConstructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	resp := &types.ConstructionPreprocessResponse{
		Options: make(map[string]interface{}),
	}
	payerAddr := request.Metadata[util.PAYER].(string)
	resp.Options[util.PAYER] = payerAddr
	var fromAddr, toAddr, fromAmount, toAmount, fromSymbol, toSymbol string
	var fromDecimals, toDecimals int32
	for _, operation := range request.Operations {
		if operation.OperationIdentifier.Index == 0 {
			fromAddr = operation.Account.Address
			fromAmount = operation.Amount.Value
			fromSymbol = operation.Amount.Currency.Symbol
			fromDecimals = operation.Amount.Currency.Decimals
		}
		if operation.OperationIdentifier.Index == 1 {
			for _, relatedOperation := range operation.RelatedOperations {
				if relatedOperation.Index == 0 {
					continue
				}
			}
			gasprice := operation.Metadata[util.GAS_PRICE]
			var ok bool
			gasPrice, ok := gasprice.(float64)
			if !ok {
				return resp, PARSE_GAS_PRICE_ERORR
			}
			gaslimit := operation.Metadata[util.GAS_LIMIT]
			gasLimit, ok := gaslimit.(float64)
			if !ok {
				return resp, PARSE_LIMIT_PRICE_ERORR
			}
			resp.Options[util.GAS_PRICE] = gasPrice
			resp.Options[util.GAS_LIMIT] = gasLimit
			toAddr = operation.Account.Address
			toAmount = operation.Amount.Value
			toSymbol = operation.Amount.Currency.Symbol
			toDecimals = operation.Amount.Currency.Decimals
		}
	}
	if fromSymbol != toSymbol || fromDecimals != toDecimals || fromAmount[1:] != toAmount {
		return resp, PARAMS_ERROR
	}
	resp.Options[util.FROM_ADDR] = fromAddr
	resp.Options[util.TO_ADDR] = toAddr
	resp.Options[util.SYMBOL] = fromSymbol
	resp.Options[util.DECIMALS] = fromDecimals
	resp.Options[util.AMOUNT] = toAmount
	return resp, nil
}

func NewConstructionAPIService(network *types.NetworkIdentifier, store *db.Store) server.ConstructionAPIServicer {
	return &ConstructionAPIService{network: network, store: store}
}

//Get Transaction Construction Metadata. endpoint:/construction/metadata
func (c ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	resp := &types.ConstructionMetadataResponse{
		Metadata: make(map[string]interface{}),
	}
	_, ok := request.Options[util.TRANSFER]
	if !ok {
		return resp, PARAMS_ERROR
	}
	resp.Metadata[util.GAS_PRICE] = "default gas price 2500,data type string"
	resp.Metadata[util.GAS_LIMIT] = "default gas limit 2000,data type string"
	resp.Metadata[util.PAYER] = "default from address,data type string"
	resp.Metadata[util.FROM_ADDR] = "from address,data type string"
	resp.Metadata[util.TO_ADDR] = "to address,data type string"
	resp.Metadata[util.AMOUNT] = "amount,data type string"
	resp.Metadata[util.ASSET] = "ont or ong,data type string"
	return resp, nil
}

//Submit a Signed Transaction .endpoint: /construction/submit
func (c ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.ConstructionSubmitResponse, *types.Error) {
	//ni := request.NetworkIdentifier
	txStr := request.SignedTransaction
	if len(txStr) == 0 {
		return nil, SIGNED_TX_INVALID
	}

	txbytes, err := common.HexToBytes(txStr)
	if err != nil {
		log.RosettaLog.Errorf("[ConstructionSubmit]HexToBytes failed:%s", err.Error())
		return nil, SIGNED_TX_INVALID
	}
	txn, err := ctypes.TransactionFromRawBytes(txbytes)
	if err != nil {
		log.RosettaLog.Errorf("[ConstructionSubmit]TransactionFromRawBytes failed:%s", err.Error())
		return nil, SIGNED_TX_INVALID
	}
	if errCode, desc := bcomn.SendTxToPool(txn); errCode != ontErrors.ErrNoError {
		log.RosettaLog.Errorf("[ConstructionSubmit]SendTxToPool failed:%s", desc)
		return nil, COMMIT_TX_FAILED
	}

	txhash := txn.Hash()

	return &types.ConstructionSubmitResponse{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: txhash.ToHexString()},
		Metadata:              nil,
	}, nil
}
