package config

import (
	"encoding/json"
	"github.com/coinbase/rosetta-sdk-go/types"
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
	Rosetta               *RosettaConfig `json:"rosetta"`
	OntologVersion        string         `json:"ontologyVersion"`
	MonitorOEP4ScriptHash []string       `json:"monitorOEP4ScriptHash"`
}

type RosettaConfig struct {
	Version string `json:"version"`
	Port    string `json:"port"`
}

func InitConfig() {
	cfile, err := os.Open("./rosetta-config.json")
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
}
