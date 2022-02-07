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

package chain

import (
	"fmt"
	"math/big"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/vm/neovm"
	"github.com/ontio/ontology/vm/neovm/errors"
	"github.com/ontio/ontology/vm/neovm/types"
)

var (
	nilAddr = common.Address{}
)

// Transfer represents a transfer within an Ontology transaction. The field
// ordering must match the internal parameter order.
type Transfer struct {
	Payer  common.Address
	From   common.Address
	To     common.Address
	Amount *big.Int
}

// ParsePayload processes the given transaction payload for transfer operations.
func ParsePayload(code []byte) ([]*Transfer, common.Address, error) {
	e := neovm.NewExecutor(code, neovm.VmFeatureFlag{})
	if err := e.Execute(); err != errors.ERR_NOT_SUPPORT_OPCODE {
		return nil, nilAddr, fmt.Errorf("chain: failed to parse payload: %s", err)
	}
	opcode := neovm.OpCode(e.Context.Code[e.Context.OpReader.Position()-1])
	switch opcode {
	case neovm.APPCALL:
		return parseApp(e)
	case neovm.SYSCALL:
		return parseSys(e.EvalStack)
	default:
		return nil, nilAddr, fmt.Errorf("chain: unexpected opcode: %v", opcode)
	}
}

func parseApp(e *neovm.Executor) ([]*Transfer, common.Address, error) {
	var contract common.Address
	err := e.Context.OpReader.ReadBytesInto(contract[:])
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: failed to read contract address: %s", err)
	}
	s := e.EvalStack
	if contract == nilAddr {
		raw, err := s.PopAsBytes()
		if err != nil {
			return nil, nilAddr, fmt.Errorf("chain: failed to get contract address: %s", err)
		}
		contract, err = common.AddressParseFromBytes(raw)
		if err != nil {
			return nil, nilAddr, fmt.Errorf("chain: unable to parse contract address: %s", err)
		}
	}
	meth, err := s.PopAsBytes()
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: failed to get method: %s", err)
	}
	xs, err := s.PopAsArray()
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: failed to get contract params: %s", err)
	}
	params := xs.Data
	switch string(meth) {
	case "transfer":
		if len(params) != 3 {
			return nil, nilAddr, fmt.Errorf("chain: unexpected transfer params length: %d", len(params))
		}
		xfer, err := parseTransferFields(params)
		if err != nil {
			return nil, nilAddr, err
		}
		return []*Transfer{xfer}, contract, nil
	case "transferFrom":
		if len(params) != 4 {
			return nil, nilAddr, fmt.Errorf("chain: unexpected transferFrom params length: %d", len(params))
		}
		xfer, err := parseTransferFromFields(params)
		if err != nil {
			return nil, nilAddr, err
		}
		return []*Transfer{xfer}, contract, nil
	case "transferMulti":
		if len(params) != 1 {
			return nil, nilAddr, fmt.Errorf("chain: unexpected transferMulti params length: %d", len(params))
		}
		xfers, err := parseAppTransferMulti(params)
		return xfers, contract, err
	default:
		return nil, nilAddr, fmt.Errorf("chain: unknown method: %s", string(meth))
	}
}

func parseAppTransferMulti(params []types.VmValue) ([]*Transfer, error) {
	xs, err := params[0].AsArrayValue()
	if err != nil {
		return nil, fmt.Errorf("chain: failed to get transferMulti internal params: %s", err)
	}
	return parseTransfers(xs.Data)
}

func parseSys(s *neovm.ValueStack) ([]*Transfer, common.Address, error) {
	_, err := s.PopAsBytes() // ignore the version value
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: invalid params: %s", err)
	}
	raw, err := s.PopAsBytes()
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: failed to get contract address: %s", err)
	}
	contract, err := common.AddressParseFromBytes(raw)
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: unable to parse contract address: %s", err)
	}
	meth, err := s.PopAsBytes()
	if err != nil {
		return nil, nilAddr, fmt.Errorf("chain: failed to get method: %s", err)
	}
	switch string(meth) {
	case "transfer", "transferV2":
		xfers, err := parseSysTransfers(s)
		return xfers, contract, err
	case "transferFrom", "transferFromV2":
		xfer, err := parseSysTransferFrom(s)
		if err != nil {
			return nil, contract, err
		}
		return []*Transfer{xfer}, contract, nil
	default:
		return nil, nilAddr, fmt.Errorf("chain: unknown method: %s", string(meth))
	}
}

func parseSysTransfers(s *neovm.ValueStack) ([]*Transfer, error) {
	xs, err := s.PopAsArray()
	if err != nil {
		return nil, fmt.Errorf("chain: failed to get contract params: %s", err)
	}
	return parseTransfers(xs.Data)
}

func parseSysTransferFrom(s *neovm.ValueStack) (*Transfer, error) {
	xs, err := s.PopAsStruct()
	if err != nil {
		return nil, fmt.Errorf("chain: failed to get contract params: %s", err)
	}
	if len(xs.Data) != 4 {
		return nil, fmt.Errorf("chain: unexpected transferFrom params length: %d", len(xs.Data))
	}
	return parseTransferFromFields(xs.Data)
}

func parseTransferFields(data []types.VmValue) (*Transfer, error) {
	raw, err := data[0].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("chain: invalid from field: %s", err)
	}
	from, err := common.AddressParseFromBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("chain: unable to parse from field: %s", err)
	}
	raw, err = data[1].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("chain: invalid to field: %s", err)
	}
	to, err := common.AddressParseFromBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("chain: unable to parse to field: %s", err)
	}
	amount, err := data[2].AsBigInt()
	if err != nil {
		return nil, fmt.Errorf("chain: invalid amount field: %s", err)
	}
	return &Transfer{
		Amount: amount,
		From:   from,
		To:     to,
	}, nil
}

func parseTransferFromFields(data []types.VmValue) (*Transfer, error) {
	raw, err := data[0].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("chain: invalid payer field: %s", err)
	}
	payer, err := common.AddressParseFromBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("chain: unable to parse payer field: %s", err)
	}
	xfer, err := parseTransferFields(data[1:])
	if err != nil {
		return nil, err
	}
	xfer.Payer = payer
	return xfer, nil
}

func parseTransfers(params []types.VmValue) ([]*Transfer, error) {
	xfers := make([]*Transfer, len(params))
	for i, data := range params {
		fields, err := data.AsStructValue()
		if err != nil {
			return nil, fmt.Errorf("chain: invalid transfer struct: %s", err)
		}
		if len(fields.Data) != 3 {
			return nil, fmt.Errorf("chain: unexpected transfer struct field length: %d", len(fields.Data))
		}
		xfer, err := parseTransferFields(fields.Data)
		if err != nil {
			return nil, err
		}
		xfers[i] = xfer
	}
	return xfers, nil
}
