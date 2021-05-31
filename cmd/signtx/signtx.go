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

// Command signtx creates a signature with an ed25519 key.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println("Usage: signtx <hex-encoded-ed25519-private-key> <hex-encoded-data-to-sign>")
		os.Exit(1)
	}
	key, err := hex.DecodeString(args[0])
	if err != nil {
		fmt.Printf("!! Failed to decode ed25519 private key: %s", err)
		os.Exit(1)
	}
	data, err := hex.DecodeString(args[1])
	if err != nil {
		fmt.Printf("!! Failed to decode data to sign: %s", err)
		os.Exit(1)
	}
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(key), data)
	fmt.Println(hex.EncodeToString(sig))
}
