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
package config

import (
	"encoding/json"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
)

const OP_TYPE_TRANSFER = "transfer"

var (
	STATUS_SUCCESS = &types.OperationStatus{
		Status:     "SUCCESS",
		Successful: true,
	}
	STATUS_FAILED = &types.OperationStatus{
		Status:     "FAILED",
		Successful: false,
	}
	Conf = &Config{}
)

type Config struct {
	Rosetta *RosettaConfig `json:"rosetta"`
	//OntologVersion        string         `json:"ontologyVersion"`
	MonitorOEP4ScriptHash []string `json:"monitorOEP4ScriptHash"`
}

type RosettaConfig struct {
	Version string `json:"version"`
	Port    int32  `json:"port"`
}

const (
	ONTOLOGY_VERSION = "1.9.0"
	ROSETTA_CONFIG   = "./rosetta-config.json"
)

var (
	RosettaConfigFlag = cli.StringFlag{
		Name:  "rosetta-config",
		Value: ROSETTA_CONFIG,
		Usage: "rosetta config `<file>`. If doesn't specifies, use ./rosetta-config.json as default",
	}
)

func InitConfig(ctx *cli.Context) {

	path := ctx.GlobalString(utils.GetFlagName(RosettaConfigFlag))
	cfile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer cfile.Close()

	jsonbytes, err := ioutil.ReadAll(cfile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(jsonbytes, Conf)
	if err != nil {
		panic(err)
	}
	if Conf.Rosetta == nil || Conf.Rosetta.Port > 65535 || Conf.Rosetta.Port <= 0 {
		panic("Conf is invalid")
	}
}
