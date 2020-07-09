# ontology-rosetta
Ontology node which follows [Rosetta](https://github.com/coinbase/rosetta-specifications) BlockChain Standard


## Build docker image

```sh
make docker
```

## Running docker image

There are two volumens to mount into the ontology-rosetta container, one is for saving blocks, the other is for config file.

```sh
# please make sure you have enough disk space for Chain dir
mkdir Chain
# you are using the default config in this repo
docker run --name ont-rosetta -d -v $(realpath Log):/data/Log -v $(realpath Chain):/data/Chain -v $(realpath rosetta-config.json):/data/rosetta-config.json -p 9090:8080 ontology-rosetta:latest
```
If you want to connect to testnet, set env NETWORK\_ID value to 2.
```sh
docker run --name ont-rosetta -d --env NETWORK_ID=2 -v $(realpath Log):/data/Log -v $(realpath Chain):/data/Chain -v $(realpath rosetta-config.json):/data/rosetta-config.json -p 9090:8080 ontology-rosetta:latest
```

## How to use

### configuration

The default configuration file is rosetta-config.json

```json
{
  "rosetta":{
    "version": "1.4.1",
    "port": 8080,
    "block_wait_time": 1
  },

  "monitorOEP4ScriptHash": []

}
```

* rosetta
  * version : rosetta sdk version
  * port: rosetta restful api port
  * block_wait_time : rosetta compute historical balance block wait time
* monitorOEP4ScriptHash:
  * OEP4 token codehash to monitor


## Restful API

Based on rosetta protocol, ontology-rosetta node provides following Restful APIs:

### Network

**/network/list**

*Get List of Available Networks*

Request:

```json
{
    "metadata": {}
}
```

Response:

Sample

```json
{
    "network_identifiers": [
        {
            "blockchain": "ont",
            "network": "mainnet"
        }
    ]
}
```

**/network/options**

*Get Network Options*

Request:

Use the available "network_identifier" from /network/list

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        }

}
```

Response:

Sample

```json
{
    "version": {
        "rosetta_version": "1.3.1",
        "node_version": "1.9.0"
    },
    "allow": {
        "operation_statuses": [
            {
                "status": "SUCCESS",
                "successful": true
            },
            {
                "status": "FAILED",
                "successful": false
            }
        ],
        "operation_types": [
            "transfer"
        ],
        "errors": [
            {
                "code": 400,
                "message": "network identifier is not supported",
                "retriable": false
            },
            {
                "code": 401,
                "message": "block identifier is empty",
                "retriable": false
            },
            {
                "code": 402,
                "message": "block index is invalid",
                "retriable": false
            },
            {
                "code": 403,
                "message": "get block failed",
                "retriable": true
            },
            {
                "code": 404,
                "message": "block hash is invalid",
                "retriable": false
            },
            {
                "code": 405,
                "message": "get transaction failed",
                "retriable": true
            },
            {
                "code": 406,
                "message": "transaction hash is invalid",
                "retriable": false
            },
            {
                "code": 407,
                "message": "commit transaction failed",
                "retriable": false
            },
            {
                "code": 408,
                "message": "tx hash is invalid",
                "retriable": false
            },
            {
                "code": 409,
                "message": "block is not exist",
                "retriable": false
            },
            {
                "code": 500,
                "message": "service not realize",
                "retriable": false
            },
            {
                "code": 501,
                "message": "addr is invalid",
                "retriable": true
            },
            {
                "code": 502,
                "message": "get balance error",
                "retriable": true
            },
            {
                "code": 503,
                "message": "parse int error",
                "retriable": true
            },
            {
                "code": 504,
                "message": "json marshal failed",
                "retriable": false
            },
            {
                "code": 505,
                "message": "parse tx payload failed",
                "retriable": false
            },
            {
                "code": 506,
                "message": "currency not config",
                "retriable": false
            },
            {
                "code": 507,
                "message": "params error",
                "retriable": true
            },
            {
                "code": 508,
                "message": "contract addr invalid",
                "retriable": true
            },
            {
                "code": 509,
                "message": "preExecute contract failed",
                "retriable": false
            },
            {
                "code": 510,
                "message": "query balance failed",
                "retriable": true
            }
        ]
    }
}
```



**/network/status**

*Get Network Status*

Request:

Use the available "network_identifier" from /network/list

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        }

}
```

Response:

Sample

```json
{
    "current_block_identifier": {
        "index": 4789126,
        "hash": "76fcf0fbd5e979721fe52e472ac79eb26f4bc502c371508574c0e03386be20e6"
    },
    "current_block_timestamp": 1560312815000,
    "genesis_block_identifier": {
        "index": 0,
        "hash": "1b8fa7f242d0eeb4395f89cbb59e4c29634047e33245c4914306e78a88e14ce5"
    },
    "peers": [
        {
            "peer_id": "000000000000000000000000b41fe9ceaaaa4d7b",
            "metadata": {
                "address": "40.113.237.243:20338",
                "height": 8454242
            }
        },
        {
            "peer_id": "000000000000000000000000a4f0c524d8efd6a8",
            "metadata": {
                "address": "139.219.141.104:20338",
                "height": 8454242
            }
        },
        {
            "peer_id": "0000000000000000000000008e6528f4659f3112",
            "metadata": {
                "address": "50.18.219.74:20338",
                "height": 8454242
            }
        }
    ]
}
```



### Account

**/account/balance**

*Get an Account Balance*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "account_identifier": {
        "address": "AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK",
        "metadata": {}
    },
	"block_identifier": {
            "index": 310
        }
}
```

Response:

Sample:

```json
{
    "block_identifier": {
        "index": 310,
        "hash": "11405500403779cff364803bbd7fe4dc74ba9119015fd79473c188b727769c52"
    },
    "balances": [
        {
            "value": "14700000",
            "currency": {
                "symbol": "ONT",
                "decimals": 0,
                "metadata": {
                    "ContractAddress": "0100000000000000000000000000000000000000",
                    "TokenType": "Governance Token"
                }
            }
        },
        {
            "value": "1750000140000",
            "currency": {
                "symbol": "ONG",
                "decimals": 9,
                "metadata": {
                    "ContractAddress": "0200000000000000000000000000000000000000",
                    "TokenType": "Utility Token"
                }
            }
        }
    ]
}
```



### Block

**/block**

*Get a Block*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "block_identifier": {
        "index":54
    }

}
```

Response:

Sample:

```json
{
    "block": {
        "block_identifier": {
            "index": 54,
            "hash": "790ea8942e5722c75ba638312caa8c1380c41da4c145d6493ae510eb6017c5f3"
        },
        "parent_block_identifier": {
            "index": 53,
            "hash": "2b52c7fcdbdcd362211e1646fa6351c8f6fd4cbfa520fe7857133e59061ff348"
        },
        "timestamp": 1530389834000,
        "transactions": [
            {
                "transaction_identifier": {
                    "hash": "20247d9df50d830b8978a5c49313a6f8a118fd5bb9c2950e3c7f95f5ac6410f6"
                },
                "operations": [
                    {
                        "operation_identifier": {
                            "index": 0
                        },
                        "type": "transfer",
                        "status": "SUCCESS",
                        "account": {
                            "address": "AJMFNZL5jGjZJEhBrJfVLHJeJ3KwiczJ6B"
                        },
                        "amount": {
                            "value": "-1000000000",
                            "currency": {
                                "symbol": "ONT",
                                "decimals": 0,
                                "metadata": {
                                    "ContractAddress": "0100000000000000000000000000000000000000",
                                    "TokenType": "Governance Token"
                                }
                            }
                        }
                    },
                    {
                        "operation_identifier": {
                            "index": 1
                        },
                        "related_operations": [
                            {
                                "index": 0
                            }
                        ],
                        "type": "transfer",
                        "status": "SUCCESS",
                        "account": {
                            "address": "AWyEMxiLUVr5MeVJe3Fw5Xsij7iZUmfYyk"
                        },
                        "amount": {
                            "value": "1000000000",
                            "currency": {
                                "symbol": "ONT",
                                "decimals": 0,
                                "metadata": {
                                    "ContractAddress": "0100000000000000000000000000000000000000",
                                    "TokenType": "Governance Token"
                                }
                            }
                        }
                    }
                ]
            }
        ]
    }
}
```



**/block/transaction**

*Get a Block Transaction*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "block_identifier": {
            "index": 54,
            "hash": "790ea8942e5722c75ba638312caa8c1380c41da4c145d6493ae510eb6017c5f3"
        },
    "transaction_identifier": {
        "hash": "20247d9df50d830b8978a5c49313a6f8a118fd5bb9c2950e3c7f95f5ac6410f6"
    }

}
```

Response:

Sample:

```json
{
    "transaction": {
        "transaction_identifier": {
            "hash": "20247d9df50d830b8978a5c49313a6f8a118fd5bb9c2950e3c7f95f5ac6410f6"
        },
        "operations": [
            {
                "operation_identifier": {
                    "index": 0
                },
                "type": "transfer",
                "status": "SUCCESS",
                "account": {
                    "address": "AJMFNZL5jGjZJEhBrJfVLHJeJ3KwiczJ6B"
                },
                "amount": {
                    "value": "-1000000000",
                    "currency": {
                        "symbol": "ONT",
                        "decimals": 0,
                        "metadata": {
                            "ContractAddress": "0100000000000000000000000000000000000000",
                            "TokenType": "Governance Token"
                        }
                    }
                }
            },
            {
                "operation_identifier": {
                    "index": 1
                },
                "related_operations": [
                    {
                        "index": 0
                    }
                ],
                "type": "transfer",
                "status": "SUCCESS",
                "account": {
                    "address": "AWyEMxiLUVr5MeVJe3Fw5Xsij7iZUmfYyk"
                },
                "amount": {
                    "value": "1000000000",
                    "currency": {
                        "symbol": "ONT",
                        "decimals": 0,
                        "metadata": {
                            "ContractAddress": "0100000000000000000000000000000000000000",
                            "TokenType": "Governance Token"
                        }
                    }
                }
            }
        ]
    }
}
```



### Construction

****/construction/derive**

*Derive Address from Public Key*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "public_key":{
        "hex_bytes":"<pubkey hex string>",
        "curev_type":"secp256k1|edwards25519",
        "metadata":{
            "type":"hex|base58"
        }
    }
}
```

Address type supports ```hex```  or ```base58``` format

Response:

Sample

```json
{
    "address":"<address>",
    "metadata":{
         "type":"hex|base58"
    }
}
```



**/construction/preprocess**

*Create Metadata Request*

Request:

```

```



Response:

Sample

```

```



**/construction/metadata**

*Create Metadata Request*

Request:

```

```



Response:

Sample

```

```



**/construction/payloads**

*Create Metadata Request*

Request:

```

```



Response:

Sample

```

```



**/construction/parse**

*Create Metadata Request*

Request:

```

```



Response:

Sample

```

```



**/construction/combine**

*Create Metadata Request*

example:  account ```AGc9NrdF5MuMJpkFfZ3MWKa67ds6H2fzud``` transfer 1 ont to account ```Af6xrG7WB9wUKQ3aRDXnfba2G5DXjqejMS``` and  ```Af6xrG7WB9wUKQ3aRDXnfba2G5DXjqejMS``` will pay for the transfer fee as payer

Request:

```json
{
	    "network_identifier":  {
            "blockchain": "ont",
            "network": "testnet"
        },
        "unsigned_transaction":"00d1594606d2c409000000000000204e000000000000ffe723aefd01bac311d8b16ff8bfd594d77f31ee7100c66b14092118e0112274581b60dfb6fedcbfdcfc044be76a7cc814ffe723aefd01bac311d8b16ff8bfd594d77f31ee6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b650000",
        "signatures":[
        		{
	        	"signing_payload":{
	        		"address":"Af6xrG7WB9wUKQ3aRDXnfba2G5DXjqejMS",
	        		"hex_bytes":"2b371f76afde8a543fd0a6a58f2578281b3517e96c2a811114ea4c78e362b221",
	        		"signature_type":"ecdsa"
	        	},
	        	"public_key":{
	        		"hex_bytes":"02263e2e1eecf7a45f21e9e0f865510966d4e93551d95876ecb3c42acf2b68aaae",
	        		"curve_type":"secp256k1"
	        	},
	        	"signature_type":"ecdsa",
	        	"hex_bytes":"3b52bc592bbba306ca9368e2808d6eb1d14fe0c3e2c801294bf8ebe3a994b464e6888038b6411a78428f9020b9f43c9dbcada7f77c0307b3ce9a410d8d2b6fa6"
        	},
        	{
	        	"signing_payload":{
	        		"address":"AGc9NrdF5MuMJpkFfZ3MWKa67ds6H2fzud",
	        		"hex_bytes":"2b371f76afde8a543fd0a6a58f2578281b3517e96c2a811114ea4c78e362b221",
	        		"signature_type":"ecdsa"
	        	},
	        	"public_key":{
	        		"hex_bytes":"03944e3ff777b14add03a76fd6767aaf4a65c227ec201375d9118d4e6b272494c7",
	        		"curve_type":"secp256k1"
	        	},
	        	"signature_type":"ecdsa",
	        	"hex_bytes":"a6f29359a94db9725ceafa37012abd3a02cff41fe1b3ca6fb0f4c58e86cd2e214567a5f29682cd4432404ecb8ded644bfb9324fe0eb746fe53097ffed13d11b1"
        	}
        ]
}
```



Response:

Sample

```json
{
    "signed_transaction": "00d1594606d2c409000000000000204e000000000000ffe723aefd01bac311d8b16ff8bfd594d77f31ee7100c66b14092118e0112274581b60dfb6fedcbfdcfc044be76a7cc814ffe723aefd01bac311d8b16ff8bfd594d77f31ee6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b65000241403b52bc592bbba306ca9368e2808d6eb1d14fe0c3e2c801294bf8ebe3a994b464e6888038b6411a78428f9020b9f43c9dbcada7f77c0307b3ce9a410d8d2b6fa6232102263e2e1eecf7a45f21e9e0f865510966d4e93551d95876ecb3c42acf2b68aaaeac4140a6f29359a94db9725ceafa37012abd3a02cff41fe1b3ca6fb0f4c58e86cd2e214567a5f29682cd4432404ecb8ded644bfb9324fe0eb746fe53097ffed13d11b1232103944e3ff777b14add03a76fd6767aaf4a65c227ec201375d9118d4e6b272494c7ac"
}
```



**/construction/metadata**

*Get Transaction Construction Metadata*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
     "options": {}
}
```

Response:

Sample

```json
{
    "metadata": {
        "calcul_history_block_height": 3627077,
        "current_block_hash": "832ed41b4e79641288ea8cd341b7949ee4773c8abb8288f4386422f9248df911",
        "current_block_height": 3627179
    }
}
```

- calcul_history_block_height:  current account balance calculate block height.


- current_block_hash: current block hash.

- current_block_height: current block height.

  ​

**/construction/submit**

*Submit a Signed Transaction*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
     "signed_transaction": "<signed tx hex>"
}
```

Response:

Sample

```json
{
    "transaction_identifier": {
        "hash": "<tx hash>"
    },
    "metadata": {}
}
```

**/construction/derive**

*Derive Address from Public Key*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "public_key":{
        "hex_bytes":"<pubkey hex string>",
        "curev_type":"secp256k1|edwards25519",
        "metadata":{
            "type":"hex|base58"
        }
    }
}
```

Address type supports ```hex```  or ```base58``` format

Response:

Sample

```json
{
    "address":"<address>",
    "metadata":{
         "type":"hex|base58"
    }
}
```



**/construction/preprocess**

*Create Metadata Request*

Request:

```

```



Response:

Sample

```

```













### Mempool

**/mempool**

*Get All Mempool Transactions*

Request:

```json
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        }
}
```

Response:

Sample

```
{
    "transaction_identifiers": [
        {
            "hash": "<tx hash>"
        }
    ]
}
```



**/mempool/transaction**

*Get a Mempool Transaction*

Request:

```
{
    "network_identifier":  {
            "blockchain": "ont",
            "network": "mainnet"
        },
    "transaction_identifier": {
        "hash": "20247d9df50d830b8978a5c49313a6f8a118fd5bb9c2950e3c7f95f5ac6410f6"
    }

}
```

Response:

Sample

```
{
    "transaction": {
        "transaction_identifier": {
            "hash": "20247d9df50d830b8978a5c49313a6f8a118fd5bb9c2950e3c7f95f5ac6410f6"
        },
        "operations": [
            {
                "operation_identifier": {
                    "index": 0
                },
                "type": "transfer",
                "status": "SUCCESS",
                "account": {
                    "address": "AJMFNZL5jGjZJEhBrJfVLHJeJ3KwiczJ6B"
                },
                "amount": {
                    "value": "-1000000000",
                    "currency": {
                        "symbol": "ONT",
                        "decimals": 0,
                        "metadata": {
                            "ContractAddress": "0100000000000000000000000000000000000000",
                            "TokenType": "Governance Token"
                        }
                    }
                }
            },
            {
                "operation_identifier": {
                    "index": 1
                },
                "related_operations": [
                    {
                        "index": 0
                    }
                ],
                "type": "transfer",
                "status": "SUCCESS",
                "account": {
                    "address": "AWyEMxiLUVr5MeVJe3Fw5Xsij7iZUmfYyk"
                },
                "amount": {
                    "value": "1000000000",
                    "currency": {
                        "symbol": "ONT",
                        "decimals": 0,
                        "metadata": {
                            "ContractAddress": "0100000000000000000000000000000000000000",
                            "TokenType": "Governance Token"
                        }
                    }
                }
            }
        ]
    }
}
```

