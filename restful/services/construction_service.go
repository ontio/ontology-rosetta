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
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-rosetta/utils"
	"github.com/ontio/ontology/core/signature"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/ontio/ontology-rosetta/common"
	db "github.com/ontio/ontology-rosetta/store"
	"github.com/ontio/ontology/common"
	ctypes "github.com/ontio/ontology/core/types"
	ontErrors "github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/http/base/actor"
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
	if signs == nil || len(signs) == 0 {
		return nil, NO_SIGS_ERROR
	}

	//todo how to solve multi-sign addr case
	ontsigns := make([]ctypes.Sig, len(signs))
	for i, s := range signs {
		pk, err := utils.TransformPubkey(s.PublicKey)
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
	meta := req.Metadata
	bts := pubkey.Bytes

	pk, err := keypair.DeserializePublicKey(bts)
	if err != nil {
		return nil, PUBKEY_HEX_ERROR
	}
	addr := ctypes.AddressFromPubKey(pk)

	resp := new(types.ConstructionDeriveResponse)

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
	context.Context,
	*types.ConstructionHashRequest,
) (*types.ConstructionHashResponse, *types.Error) {
	panic("implement me")
}

func (c ConstructionAPIService) ConstructionParse(
	context.Context,
	*types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	panic("implement me")
}

func (c ConstructionAPIService) ConstructionPayloads(
	context.Context,
	*types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	panic("implement me")
}

func (c ConstructionAPIService) ConstructionPreprocess(
	context.Context,
	*types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	panic("implement me")
}

func NewConstructionAPIService(network *types.NetworkIdentifier, store *db.Store) server.ConstructionAPIServicer {
	return &ConstructionAPIService{network: network, store: store}
}

//Get Transaction Construction Metadata. endpoint:/construction/metadata
func (c ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	//todo define the options
	//ni := request.NetworkIdentifier
	//opt := request.Options

	//return the current block height and hash
	//following by the example
	height := actor.GetCurrentBlockHeight()
	hash := actor.CurrentBlockHash()

	metadata := make(map[string]interface{})
	metadata["current_block_hash"] = hash.ToHexString()
	metadata["current_block_height"] = height
	historyHeight, err := getHeightFromStore(c.store)
	if err != nil {
		log.RosettaLog.Errorf("getHeightFromStore err:%s", err)
	} else {
		metadata["calcul_history_block_height"] = historyHeight
	}
	resp := &types.ConstructionMetadataResponse{Metadata: metadata}

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
