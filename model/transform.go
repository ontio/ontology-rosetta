/*
 * Copyright (C) 2021 The ontology Authors
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

// Package model implements the data model for use in the internal data store.
package model

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ontio/ontology/common"
)

func (b *Block) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		fmt.Fprintf(f, "{timestamp: %d, transactions: [", b.Timestamp)
		for i, txn := range b.Transactions {
			if i != 0 {
				fmt.Fprint(f, ", ")
			}
			txn.Format(f, 's')
		}
		fmt.Fprint(f, "]}")
	}
}

func (t *Transaction) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		fmt.Fprint(f, "{hash: ")
		f.Write(hexaddr(t.Hash))
		fmt.Fprint(f, ", transfers: [")
		for i, xfer := range t.Transfers {
			if i != 0 {
				fmt.Fprint(f, ", ")
			}
			xfer.Format(f, 's')
		}
		fmt.Fprint(f, "]}")
	}
}

func (t *Transfer) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		amount := (&big.Int{}).SetBytes(t.Amount)
		fmt.Fprint(f, "{amount: ")
		fmt.Fprint(f, amount.String())
		fmt.Fprint(f, ", contract: ")
		f.Write(hexaddr(t.Contract))
		fmt.Fprint(f, ", from: ")
		f.Write(hexaddr(t.From))
		fmt.Fprint(f, ", to: ")
		f.Write(hexaddr(t.To))
		if t.IsGas {
			fmt.Fprint(f, ", is_gas: true}")
		} else {
			fmt.Fprint(f, ", is_gas: false}")
		}
	}
}

func hexaddr(addr []byte) []byte {
	src := common.ToArrayReverse(addr)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return dst
}
