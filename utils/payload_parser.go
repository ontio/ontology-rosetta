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
	"fmt"

	"github.com/ontio/ontology/v2/common"
	"github.com/ontio/ontology/v2/vm/neovm"
	"github.com/ontio/ontology/v2/vm/neovm/errors"
	"github.com/ontio/ontology/v2/vm/neovm/types"
)

type TransferState struct {
	Spender common.Address
	From    common.Address
	To      common.Address
	Value   uint64
}

func ParsePayload(code []byte) ([]*TransferState, common.Address, error) {
	executor := neovm.NewExecutor(code, neovm.VmFeatureFlag{})
	if err := executor.Execute(); err != errors.ERR_NOT_SUPPORT_OPCODE {
		return nil, common.ADDRESS_EMPTY, fmt.Errorf("parse payload failed: %s", err)
	}
	opcode := neovm.OpCode(executor.Context.Code[executor.Context.OpReader.Position()-1])
	if opcode == neovm.SYSCALL {
		return parseOntPayload(executor)
	} else if opcode == neovm.APPCALL {
		return parseOep4Payload(executor)
	} else {
		return nil, common.ADDRESS_EMPTY, fmt.Errorf("invalid opcode, %v", opcode)
	}
}

func parseOntPayload(executor *neovm.Executor) ([]*TransferState, common.Address, error) {
	// pop last nil value
	_, err := executor.EvalStack.PopAsBytes()
	if err != nil {
		return nil, common.ADDRESS_EMPTY, fmt.Errorf("invalid param format, %s", err)
	}
	contractBytes, err := executor.EvalStack.PopAsBytes()
	if err != nil {
		return nil, common.ADDRESS_EMPTY, fmt.Errorf("invalid contract addr, %s", err)
	}
	contractAddr, err := common.AddressParseFromBytes(contractBytes)
	if err != nil {
		return nil, common.ADDRESS_EMPTY, fmt.Errorf("parse contract addr, %s", err)
	}
	methodName, err := executor.EvalStack.PopAsBytes()
	if err != nil {
		return nil, contractAddr, fmt.Errorf("invalid method name, %s", err)
	}
	switch string(methodName) {
	case "transfer":
		result, err := parseOntTransferPayload(executor.EvalStack)
		return result, contractAddr, err
	case "transferFrom":
		transferFrom, err := parseOntTransferFromPayload(executor.EvalStack)
		if err != nil {
			return nil, contractAddr, err
		}
		return []*TransferState{transferFrom}, contractAddr, nil
	default:
		return nil, contractAddr, fmt.Errorf("illegal method name: %s", string(methodName))
	}
}

func parseOep4Payload(executor *neovm.Executor) ([]*TransferState, common.Address, error) {
	var address common.Address
	err := executor.Context.OpReader.ReadBytesInto(address[:])
	if err != nil {
		return nil, address, fmt.Errorf("read contract addr, %s", err)
	}
	if address == common.ADDRESS_EMPTY {
		addrBytes, err := executor.EvalStack.PopAsBytes()
		if err != nil {
			return nil, address, fmt.Errorf("pop contract addr, %s", err)
		}
		address, err = common.AddressParseFromBytes(addrBytes)
		if err != nil {
			return nil, address, fmt.Errorf("parse contract addr, %s", err)
		}
	}
	methodName, err := executor.EvalStack.PopAsBytes()
	if err != nil {
		return nil, address, fmt.Errorf("invalid method name, %s", err)
	}
	switch string(methodName) {
	case "transfer":
		transfer, err := parseOep4TransferPayload(executor.EvalStack)
		if err != nil {
			return nil, address, err
		}
		return []*TransferState{transfer}, address, nil
	case "transferMulti":
		result, err := parseOep4TransferMultiPayload(executor.EvalStack)
		return result, address, err
	case "transferFrom":
		transferFrom, err := parseOep4TransferFromPayload(executor.EvalStack)
		if err != nil {
			return nil, address, err
		}
		return []*TransferState{transferFrom}, address, nil
	default:
		return nil, address, fmt.Errorf("illegal method name: %s", string(methodName))
	}
}

func parseOep4TransferPayload(stack *neovm.ValueStack) (*TransferState, error) {
	param, err := stack.PopAsArray()
	if err != nil {
		return nil, fmt.Errorf("invalid param, %s", err)
	}
	if len(param.Data) != 3 {
		return nil, fmt.Errorf("invalid param len %d", len(param.Data))
	}
	return parseTransferParam(param.Data)
}

func parseOep4TransferMultiPayload(stack *neovm.ValueStack) ([]*TransferState, error) {
	param, err := stack.PopAsArray()
	if err != nil {
		return nil, fmt.Errorf("invalid param, %s", err)
	}
	if len(param.Data) != 1 {
		return nil, fmt.Errorf("invalid param len %d", len(param.Data))
	}
	paramData, err := param.Data[0].AsArrayValue()
	if err != nil {
		return nil, fmt.Errorf("invalid param data, %s", err)
	}
	result := make([]*TransferState, 0)
	for _, data := range paramData.Data {
		param, err := data.AsStructValue()
		if err != nil {
			return nil, fmt.Errorf("invalid transfer struct, %s", err)
		}
		if len(param.Data) != 3 {
			return nil, fmt.Errorf("invalid transfer struct field len, %d", len(param.Data))
		}
		transferState, err := parseTransferParam(param.Data)
		if err != nil {
			return nil, err
		}
		result = append(result, transferState)
	}
	return result, nil
}

func parseOntTransferPayload(stack *neovm.ValueStack) ([]*TransferState, error) {
	paramArray, err := stack.PopAsArray()
	if err != nil {
		return nil, fmt.Errorf("invalid param, %s", err)
	}
	result := make([]*TransferState, 0)
	for _, data := range paramArray.Data {
		param, err := data.AsStructValue()
		if err != nil {
			return nil, fmt.Errorf("invalid transfer struct, %s", err)
		}
		if len(param.Data) != 3 {
			return nil, fmt.Errorf("invalid transfer struct field len %d", len(param.Data))
		}
		transferState, err := parseTransferParam(param.Data)
		if err != nil {
			return nil, err
		}
		result = append(result, transferState)
	}
	return result, nil
}

func parseOep4TransferFromPayload(stack *neovm.ValueStack) (*TransferState, error) {
	param, err := stack.PopAsArray()
	if err != nil {
		return nil, fmt.Errorf("invalid param, %s", err)
	}
	if len(param.Data) != 4 {
		return nil, fmt.Errorf("invalid param len %d", len(param.Data))
	}
	return parseTransferFromParam(param.Data)
}

func parseOntTransferFromPayload(stack *neovm.ValueStack) (*TransferState, error) {
	param, err := stack.PopAsStruct()
	if err != nil {
		return nil, fmt.Errorf("invalid param, %s", err)
	}
	if len(param.Data) != 4 {
		return nil, fmt.Errorf("invalid param field len %d", len(param.Data))
	}
	return parseTransferFromParam(param.Data)
}

func parseTransferParam(data []types.VmValue) (*TransferState, error) {
	fromBytes, err := data[0].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("invalid from param, %s", err)
	}
	fromAddr, err := common.AddressParseFromBytes(fromBytes)
	if err != nil {
		return nil, fmt.Errorf("parse from, %s", err)
	}
	toBytes, err := data[1].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("invalid to param, %s", err)
	}
	toAddr, err := common.AddressParseFromBytes(toBytes)
	if err != nil {
		return nil, fmt.Errorf("parse to, %s", err)
	}
	amount, err := data[2].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("invalid amount param, %s", err)
	}
	return &TransferState{
		From:  fromAddr,
		To:    toAddr,
		Value: uint64(amount),
	}, nil
}

func parseTransferFromParam(data []types.VmValue) (*TransferState, error) {
	spenderBytes, err := data[0].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("invalid spender param, %s", err)
	}
	spender, err := common.AddressParseFromBytes(spenderBytes)
	if err != nil {
		return nil, fmt.Errorf("parse spender, %s", err)
	}
	fromBytes, err := data[1].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("invalid from param, %s", err)
	}
	from, err := common.AddressParseFromBytes(fromBytes)
	if err != nil {
		return nil, fmt.Errorf("parse from, %s", err)
	}
	toBytes, err := data[2].AsBytes()
	if err != nil {
		return nil, fmt.Errorf("invalid to param, %s", err)
	}
	to, err := common.AddressParseFromBytes(toBytes)
	if err != nil {
		return nil, fmt.Errorf("parse to, %s", err)
	}
	value, err := data[3].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("invalid value param, %s", err)
	}
	return &TransferState{
		Spender: spender,
		From:    from,
		To:      to,
		Value:   uint64(value),
	}, nil
}
