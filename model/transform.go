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

func (c *ConstructOptions) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		fmt.Fprint(f, "{")
		written := false
		if len(c.Amount) > 0 {
			amount := (&big.Int{}).SetBytes(c.Amount)
			fmt.Fprint(f, "amount: ")
			fmt.Fprint(f, amount.String())
			written = true
		}
		if len(c.Contract) > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprint(f, "contract: ")
			f.Write(hexaddr(c.Contract))
			written = true
		}
		if len(c.From) > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprint(f, "from: ")
			f.Write(hexaddr(c.From))
			written = true
		}
		if c.GasLimit > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprintf(f, "gas_limit: %d", c.GasLimit)
			written = true
		}
		if c.GasPrice > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprintf(f, "gas_price: %d", c.GasPrice)
			written = true
		}
		if c.Nonce > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprintf(f, "nonce: %d", c.Nonce)
			written = true
		}
		if len(c.Payer) > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprint(f, "payer: ")
			f.Write(hexaddr(c.Payer))
			written = true
		}
		if len(c.To) > 0 {
			if written {
				fmt.Fprint(f, ", ")
			}
			fmt.Fprint(f, "to: ")
			f.Write(hexaddr(c.To))
		}
		fmt.Fprint(f, "}")
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
