# ontology-rosetta
Ontology node which follows Rosetta BlockChain Standard


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
docker run --name ont-rosetta -d -v $(realpath Chain):/data/Chain -v $(realpath rosetta-config.json):/data/rosetta-config.json -p 9090:8080 ontology-rosetta:latest
```
## How to use

### configuration

The default configuration file is rosetta-config.json

```
{
  "rosetta":{
    "version": "1.3.1",
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
  * OEP4 token codehash to monitor, you can find them on <https://explorer.ont.io/token/list/oep4/10/1>


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
            "peer_id": "4584491680478203539",
            "metadata": {
                "address": "3.21.156.220:20338",
                "height": 8325360,
                "state": 4
            }
        },
        {
            "peer_id": "15540261876814914032",
            "metadata": {
                "address": "23.99.134.190:20338",
                "height": 8325359,
                "state": 4
            }
        },
        {
            "peer_id": "18276482511736887962",
            "metadata": {
                "address": "18.220.195.232:20338",
                "height": 8325360,
                "state": 4
            }
        },
        {
            "peer_id": "16061082910762282354",
            "metadata": {
                "address": "139.219.138.225:20338",
                "height": 8325359,
                "state": 4
            }
        },
        {
            "peer_id": "6670297000549095677",
            "metadata": {
                "address": "52.231.153.200:20338",
                "height": 8325360,
                "state": 4
            }
        },
        {
            "peer_id": "407576854451017996",
            "metadata": {
                "address": "34.217.180.221:20338",
                "height": 8325360,
                "state": 4
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

