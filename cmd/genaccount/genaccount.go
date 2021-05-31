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

// Command genaccount creates the rosetta-cli config for a prefunded account.
//
// The account will need to be funded manually.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/program"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Printf("!! Failed to generate keypair: %s", err)
		os.Exit(1)
	}
	addr := common.AddressFromVmCode(program.ProgramFromPubKey(pub))
	out, err := json.Marshal(&modules.PrefundedAccount{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr.ToBase58(),
		},
		Currency: &types.Currency{
			Decimals: 9,
			Metadata: map[string]interface{}{
				"contract": "0200000000000000000000000000000000000000",
			},
			Symbol: "ONG",
		},
		CurveType:     types.Edwards25519,
		PrivateKeyHex: hex.EncodeToString(priv[:32]),
	})
	if err != nil {
		fmt.Printf("!! Failed to encode prefunded account: %s", err)
		os.Exit(1)
	}
	// NOTE(tav): We decode and re-encode the JSON so that the keys are in
	// lexicographic order.
	var val interface{}
	if err := json.Unmarshal(out, &val); err != nil {
		fmt.Printf("!! Failed to decode JSON: %s", err)
		os.Exit(1)
	}
	out, err = json.MarshalIndent(val, "", "    ")
	if err != nil {
		fmt.Printf("!! Failed to re-encode prefunded account: %s", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
