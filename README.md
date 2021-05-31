# ontology-rosetta

[Rosetta](https://github.com/coinbase/rosetta-specifications) implementation of Ontology.

## Overview

Here we lay out the procedure one needs to follow to deploy an Ontology node that adheres to Rosetta's blockchain standards. By complying to the **Rosetta blockchain specifications**, we at Ontology aim to streamline the development process for blockchain developers by ensuring certain aspects of the system are structured in a manner such that basic operations such as the **deployment process**, **communication**, and certain **data formats** are **standardized**, thus increasing the **overall flexibility** of the system.

## Build docker image

```sh
make docker
```

## Run docker image

There are two volumes to mount into the ontology-rosetta container, one is for saving blocks, the other is for the config file.

```sh
# please make sure you have enough disk space for Chain dir
mkdir Chain
# you are using the default config in this repo
docker run --name ont-rosetta -d -v $(realpath Log):/data/Log -v $(realpath Chain):/data/Chain -v $(realpath server-config.json):/data/server-config.json -p 9090:8080 ontology-rosetta:latest
```
If you want to connect to testnet, set env `NETWORK_ID` value to `2`.
```sh
docker run --name ont-rosetta -d --env NETWORK_ID=2 -v $(realpath Log):/data/Log -v $(realpath Chain):/data/Chain -v $(realpath server-config.json):/data/server-config.json -p 9090:8080 ontology-rosetta:latest
```

## Config

The default config file is `server-config.json`:

```json
{
  "block_wait_seconds": 1,
  "oep4_tokens": [],
  "port": 8080
}
```

Objects within the `oep4_tokens` array must follow this structure:

```json
{
  "contract": "ff31ec74d01f7b7d45ed2add930f5d2239f7de33",
  "decimals": 9,
  "symbol": "WING"
}
```

## Rosetta API

### Network

**/network/list**

*Get List of Available Networks*

Request:

```json
{}
```

Sample Response:

```json
{
  "network_identifiers": [
    {
      "blockchain": "ontology",
      "network": "testnet"
    }
  ]
}
```

**/network/options**

*Get Network Options*

Request:

Use the `network_identifier` from `/network/list`:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  }
}
```

Sample Response:

```json
{
  "allow": {
    "balance_exemptions": null,
    "call_methods": null,
    "errors": [
      {
        "code": 101,
        "message": "method not implemented",
        "retriable": false
      },
      {
        "code": 102,
        "message": "method not available in offline mode",
        "retriable": false
      },
      {
        "code": 201,
        "message": "currency not defined",
        "retriable": true
      },
      {
        "code": 301,
        "message": "datastore error",
        "retriable": true
      },
      {
        "code": 302,
        "message": "datastore transaction conflict",
        "retriable": true
      },
      {
        "code": 303,
        "message": "datastore consistency failure",
        "retriable": true
      },
      {
        "code": 304,
        "message": "unexpected internal error",
        "retriable": true
      },
      {
        "code": 305,
        "message": "nonce generation failed",
        "retriable": true
      },
      {
        "code": 306,
        "message": "protobuf error",
        "retriable": false
      },
      {
        "code": 401,
        "message": "invalid account address",
        "retriable": false
      },
      {
        "code": 402,
        "message": "invalid block hash",
        "retriable": false
      },
      {
        "code": 403,
        "message": "invalid block identifier",
        "retriable": false
      },
      {
        "code": 404,
        "message": "invalid block index",
        "retriable": false
      },
      {
        "code": 405,
        "message": "invalid construct options",
        "retriable": false
      },
      {
        "code": 406,
        "message": "invalid contract address",
        "retriable": false
      },
      {
        "code": 407,
        "message": "invalid currency",
        "retriable": false
      },
      {
        "code": 408,
        "message": "invalid gas limit",
        "retriable": false
      },
      {
        "code": 409,
        "message": "invalid gas price",
        "retriable": false
      },
      {
        "code": 410,
        "message": "invalid nonce",
        "retriable": false
      },
      {
        "code": 411,
        "message": "invalid ops intent",
        "retriable": false
      },
      {
        "code": 412,
        "message": "invalid payer address",
        "retriable": false
      },
      {
        "code": 413,
        "message": "invalid public key",
        "retriable": false
      },
      {
        "code": 414,
        "message": "invalid request field",
        "retriable": false
      },
      {
        "code": 415,
        "message": "invalid signature",
        "retriable": false
      },
      {
        "code": 416,
        "message": "invalid transaction hash",
        "retriable": false
      },
      {
        "code": 417,
        "message": "invalid transaction payload",
        "retriable": false
      },
      {
        "code": 501,
        "message": "broadcast failed",
        "retriable": true
      },
      {
        "code": 502,
        "message": "transaction not in mempool",
        "retriable": true
      },
      {
        "code": 503,
        "message": "unknown block hash",
        "retriable": true
      },
      {
        "code": 504,
        "message": "unknown block index",
        "retriable": true
      }
    ],
    "historical_balance_lookup": true,
    "mempool_coins": false,
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
      "gas_fee",
      "transfer"
    ]
  },
  "version": {
    "node_version": "1.13.2",
    "rosetta_version": "1.13.3"
  }
}
```

**/network/status**

*Get Network Status*

Request:

Use the `network_identifier` from `/network/list`:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  }
}
```

Sample Response:

```json
{
  "current_block_identifier": {
    "hash": "16607e636debd83d4591ede226d3e4df66ce91974c0c92bb7a2b24bd41c5109a",
    "index": 16029266
  },
  "current_block_timestamp": 1622449609000,
  "genesis_block_identifier": {
    "hash": "44425ae42a394ec0c5f3e41d757ffafa790b53f7301147a291ab9b60a956394c",
    "index": 0
  },
  "peers": [
    {
      "metadata": {
        "address": "45.43.63.93:20338",
        "height": 16029266,
        "last_contact": "2021-05-31T08:27:24Z",
        "relay": true,
        "self": "49d991f2ebc3f8a98ad48c3d090276c9ff10b61e",
        "version": "v2.2.0-0-ga25aaea"
      },
      "peer_id": "551892e4fec71a49a4d63bf5177ca29072f94177"
    },
    {
      "metadata": {
        "address": "35.246.14.9:20338",
        "height": 16029266,
        "last_contact": "2021-05-31T08:27:24Z",
        "relay": true,
        "self": "49d991f2ebc3f8a98ad48c3d090276c9ff10b61e",
        "version": "v2.2.0-0-ga25aaea"
      },
      "peer_id": "d214c85303149288b16e424f50e35d13c55e710f"
    },
    {
      "metadata": {
        "address": "132.145.87.235:20338",
        "height": 1928227,
        "last_contact": "2021-05-31T08:27:26Z",
        "relay": true,
        "self": "49d991f2ebc3f8a98ad48c3d090276c9ff10b61e",
        "version": "d7f8833"
      },
      "peer_id": "000000000000000000000000e10f0e0f7f85dae9"
    }
  ],
  "sync_status": {
    "current_index": 16029266,
    "synced": true,
    "target_index": 16029266
  }
}
```

### Account

**/account/balance**

*Get an Account Balance*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "account_identifier": {
    "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
  },
  "block_identifier": {
    "index": 16028389,
    "hash": "1dc336bb7a098c3d6cdc34f7a2a96ab8e726664c5c45924d4b4865fb7c52a9a0"
  }
}
```

Sample Response:

```json
{
  "balances": [
    {
      "currency": {
        "decimals": 0,
        "metadata": {
          "contract": "0100000000000000000000000000000000000000"
        },
        "symbol": "ONT"
      },
      "value": "50"
    },
    {
      "currency": {
        "decimals": 9,
        "metadata": {
          "contract": "0200000000000000000000000000000000000000"
        },
        "symbol": "ONG"
      },
      "value": "99900000000"
    }
  ],
  "block_identifier": {
    "hash": "1dc336bb7a098c3d6cdc34f7a2a96ab8e726664c5c45924d4b4865fb7c52a9a0",
    "index": 16028389
  }
}
```

### Block

**/block**

*Get a Block*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "block_identifier": {
    "index": 83690
  }
}
```

Sample Response:

```json
{
  "block": {
    "block_identifier": {
      "hash": "513949285fdbd66c8cd40427a9832fe15002b2fbe17cf5da3746340fd922efe1",
      "index": 83690
    },
    "parent_block_identifier": {
      "hash": "a345c752d07d46b7237f36ecb7e021d26ec97c9aa78472851a185d7261fb4e95",
      "index": 83689
    },
    "timestamp": 1532688780000,
    "transactions": [
      {
        "operations": [
          {
            "operation_identifier": {
              "index": 0
            },
            "type": "transfer",
            "status": "SUCCESS",
            "account": {
              "address": "AFmseVrdL9f9oyCzZefL9tG6UbvhUMqNMV"
            },
            "amount": {
              "value": "-9438053292451530",
              "currency": {
                "symbol": "ONG",
                "decimals": 9,
                "metadata": {
                  "contract": "0200000000000000000000000000000000000000"
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
              "address": "AHmwjZ58TLsH5dhvBkAEnsZ2tY9XeDPLXD"
            },
            "amount": {
              "value": "9438053292451530",
              "currency": {
                "symbol": "ONG",
                "decimals": 9,
                "metadata": {
                  "contract": "0200000000000000000000000000000000000000"
                }
              }
            }
          },
          {
            "operation_identifier": {
              "index": 2
            },
            "type": "gas_fee",
            "status": "SUCCESS",
            "account": {
              "address": "AHmwjZ58TLsH5dhvBkAEnsZ2tY9XeDPLXD"
            },
            "amount": {
              "value": "-10000000",
              "currency": {
                "symbol": "ONG",
                "decimals": 9,
                "metadata": {
                  "contract": "0200000000000000000000000000000000000000"
                }
              }
            }
          },
          {
            "operation_identifier": {
              "index": 3
            },
            "related_operations": [
              {
                "index": 2
              }
            ],
            "type": "gas_fee",
            "status": "SUCCESS",
            "account": {
              "address": "AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK"
            },
            "amount": {
              "value": "10000000",
              "currency": {
                "symbol": "ONG",
                "decimals": 9,
                "metadata": {
                  "contract": "0200000000000000000000000000000000000000"
                }
              }
            }
          }
        ],
        "transaction_identifier": {
          "hash": "659ff28a14bac75883f0b4501fcdd34db170697773a61a8580806d0d6e5773ec"
        }
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
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "block_identifier": {
    "index": 83690,
    "hash": "513949285fdbd66c8cd40427a9832fe15002b2fbe17cf5da3746340fd922efe1"
  },
  "transaction_identifier": {
    "hash": "659ff28a14bac75883f0b4501fcdd34db170697773a61a8580806d0d6e5773ec"
  }
}
```

Sample Response:

```json
{
  "transaction": {
    "operations": [
      {
        "account": {
          "address": "AFmseVrdL9f9oyCzZefL9tG6UbvhUMqNMV"
        },
        "amount": {
          "currency": {
            "symbol": "ONG",
            "decimals": 9,
            "metadata": {
              "contract": "0200000000000000000000000000000000000000"
            }
          },
          "value": "-9438053292451530"
        },
        "operation_identifier": {
          "index": 0
        },
        "status": "SUCCESS",
        "type": "transfer"
      },
      {
        "account": {
          "address": "AHmwjZ58TLsH5dhvBkAEnsZ2tY9XeDPLXD"
        },
        "amount": {
          "currency": {
            "symbol": "ONG",
            "decimals": 9,
            "metadata": {
              "contract": "0200000000000000000000000000000000000000"
            }
          },
          "value": "9438053292451530"
        },
        "operation_identifier": {
          "index": 1
        },
        "related_operations": [
          {
            "index": 0
          }
        ],
        "status": "SUCCESS",
        "type": "transfer"
      },
      {
        "account": {
          "address": "AHmwjZ58TLsH5dhvBkAEnsZ2tY9XeDPLXD"
        },
        "amount": {
          "currency": {
            "symbol": "ONG",
            "decimals": 9,
            "metadata": {
              "contract": "0200000000000000000000000000000000000000"
            }
          },
          "value": "-10000000"
        },
        "operation_identifier": {
          "index": 2
        },
        "status": "SUCCESS",
        "type": "gas_fee"
      },
      {
        "account": {
          "address": "AFmseVrdL9f9oyCzZefL9tG6UbviEH9ugK"
        },
        "amount": {
          "currency": {
            "symbol": "ONG",
            "decimals": 9,
            "metadata": {
              "contract": "0200000000000000000000000000000000000000"
            }
          },
          "value": "10000000"
        },
        "operation_identifier": {
          "index": 3
        },
        "related_operations": [
          {
            "index": 2
          }
        ],
        "status": "SUCCESS",
        "type": "gas_fee"
      }
    ],
    "transaction_identifier": {
      "hash": "659ff28a14bac75883f0b4501fcdd34db170697773a61a8580806d0d6e5773ec"
    }
  }
}
```

### Construction

**/construction/derive**

*Derive an AccountIdentifier from a PublicKey*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "public_key": {
    "hex_bytes": "06054a5f08c5ae703c5ffc12f6c63f76dce9daeb3e32d98d96e69befbc70f3de",
    "curve_type": "edwards25519"
  }
}
```

Sample Response:

```json
{
  "account_identifier": {
    "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
  }
}
```

**/construction/preprocess**

*Create a Request to Fetch Metadata*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "operations": [
    {
      "operation_identifier": {
        "index": 0
      },
      "type": "transfer",
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "value": "-1",
        "currency": {
          "symbol": "ONT",
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
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
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "value": "1",
        "currency": {
          "symbol": "ONT",
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
          }
        }
      }
    }
  ],
  "metadata": {
  }
}
```

The request's `metadata` field supports some optional `uint32` subfields:

* `gas_limit` — If unspecified, this will default to the minimum transaction gas
  value.

* `gas_price` — If unspecified, this will default to using the current network
  gas price.

* `nonce` — If unspecified, this will default to a randomly generated nonce that
  doesn't conflict with any transactions already seen by the node.

* `payer` — If unspecified, this will default to the sender inferred from the
  provided operations.

Sample Response:

```json
{
  "options": {
    "protobuf": "0a0101121400000000000000000000000000000000000000011a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f3a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f421409fa00755de7e8fc9eafe28bbf31384b56e18e0f"
  }
}
```

**/construction/metadata**

*Get Metadata for Transaction Construction*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "options": {
    "protobuf": "0a0101121400000000000000000000000000000000000000011a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f3a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f421409fa00755de7e8fc9eafe28bbf31384b56e18e0f"
  }
}
```

Sample Response:

```json
{
  "metadata": {
    "protobuf": "0a0101121400000000000000000000000000000000000000011a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f20a09c0128c41330cbb1f4b00f3a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f421409fa00755de7e8fc9eafe28bbf31384b56e18e0f"
  }
}
```

**/construction/payloads**

*Generate an Unsigned Transaction and Signing Payloads*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "operations": [
    {
      "operation_identifier": {
        "index": 0
      },
      "type": "transfer",
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "value": "-1",
        "currency": {
          "symbol": "ONT",
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
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
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "value": "1",
        "currency": {
          "symbol": "ONT",
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
          }
        }
      }
    }
  ],
  "metadata": {
    "protobuf": "0a0101121400000000000000000000000000000000000000011a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f20a09c0128c41330cbb1f4b00f3a1409fa00755de7e8fc9eafe28bbf31384b56e18e0f421409fa00755de7e8fc9eafe28bbf31384b56e18e0f"
  }
}
```

Sample Response:

```json
{
  "payloads": [
    {
      "account_identifier": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d",
      "hex_bytes": "f33ee01563f1f93f80bbdc031ded71d4cd4a23b513fc52864cfa598f18a2eb53",
      "signature_type": "ed25519"
    }
  ],
  "unsigned_transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b650000"
}
```

**/construction/parse**

*Parse a Transaction*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "signed": false,
  "transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b650000"
}
```

Sample Response:

```json
{
  "metadata": {
    "gas_limit": 20000,
    "gas_price": 2500,
    "nonce": 4129102027,
    "payer": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
  },
  "operations": [
    {
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "currency": {
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
          },
          "symbol": "ONT"
        },
        "value": "-1"
      },
      "operation_identifier": {
        "index": 0
      },
      "type": "transfer"
    },
    {
      "account": {
        "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
      },
      "amount": {
        "currency": {
          "decimals": 0,
          "metadata": {
            "contract": "0100000000000000000000000000000000000000"
          },
          "symbol": "ONT"
        },
        "value": "1"
      },
      "operation_identifier": {
        "index": 1
      },
      "related_operations": [
        {
          "index": 0
        }
      ],
      "type": "transfer"
    }
  ]
}
```

**/construction/combine**

*Generate Network Transaction from Signatures*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "unsigned_transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b650000",
  "signatures": [
    {
      "signing_payload": {
        "account_identifier": {
          "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
        },
        "hex_bytes": "f33ee01563f1f93f80bbdc031ded71d4cd4a23b513fc52864cfa598f18a2eb53",
        "signature_type": "ed25519"
      },
      "public_key": {
        "hex_bytes": "06054a5f08c5ae703c5ffc12f6c63f76dce9daeb3e32d98d96e69befbc70f3de",
        "curve_type": "edwards25519"
      },
      "signature_type": "ed25519",
      "hex_bytes": "3e82f222223192fc65ed8b29a7016025480914cbfecd9f55de1b7e50bfed82663d296d1c5171d1af5dd5eca8b8b45287355b2ac2f0a0c378ced6b929b6f0bf02"
    }
  ]
}
```

Sample Response:

```json
{
  "signed_transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b65000142410a3e82f222223192fc65ed8b29a7016025480914cbfecd9f55de1b7e50bfed82663d296d1c5171d1af5dd5eca8b8b45287355b2ac2f0a0c378ced6b929b6f0bf022422141906054a5f08c5ae703c5ffc12f6c63f76dce9daeb3e32d98d96e69befbc70f3deac"
}
```

**/construction/hash**

*Get the Hash of a Signed Transaction*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "signed_transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b65000142410a3e82f222223192fc65ed8b29a7016025480914cbfecd9f55de1b7e50bfed82663d296d1c5171d1af5dd5eca8b8b45287355b2ac2f0a0c378ced6b929b6f0bf022422141906054a5f08c5ae703c5ffc12f6c63f76dce9daeb3e32d98d96e69befbc70f3deac"
}
```

Sample Response:

```json
{
  "transaction_identifier": {
    "hash": "53eba2188f59fa4c8652fc13b5234acdd471ed1d03dcbb803ff9f16315e03ef3"
  }
}
```

**/construction/submit**

*Submit a Signed Transaction*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "signed_transaction": "00d1cb181df6c409000000000000204e00000000000009fa00755de7e8fc9eafe28bbf31384b56e18e0f7100c66b1409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc81409fa00755de7e8fc9eafe28bbf31384b56e18e0f6a7cc8516a7cc86c51c1087472616e736665721400000000000000000000000000000000000000010068164f6e746f6c6f67792e4e61746976652e496e766f6b65000142410a3e82f222223192fc65ed8b29a7016025480914cbfecd9f55de1b7e50bfed82663d296d1c5171d1af5dd5eca8b8b45287355b2ac2f0a0c378ced6b929b6f0bf022422141906054a5f08c5ae703c5ffc12f6c63f76dce9daeb3e32d98d96e69befbc70f3deac"
}
```

Sample Response:

```json
{
  "transaction_identifier": {
    "hash": "53eba2188f59fa4c8652fc13b5234acdd471ed1d03dcbb803ff9f16315e03ef3"
  }
}
```

### Mempool

**/mempool**

*Get All Mempool Transactions*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  }
}
```

Sample Response:

```json
{
  "transaction_identifiers": ["53eba2188f59fa4c8652fc13b5234acdd471ed1d03dcbb803ff9f16315e03ef3"]
}
```

**/mempool/transaction**

*Get a Mempool Transaction*

Request:

```json
{
  "network_identifier": {
    "blockchain": "ontology",
    "network": "testnet"
  },
  "transaction_identifier": {
    "hash": "53eba2188f59fa4c8652fc13b5234acdd471ed1d03dcbb803ff9f16315e03ef3"
  }
}
```

Sample Response:

```json
{
  "transaction": {
    "operations": [
      {
        "account": {
          "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
        },
        "amount": {
          "currency": {
            "symbol": "ONT",
            "decimals": 0,
            "metadata": {
              "contract": "0100000000000000000000000000000000000000"
            }
          },
          "value": "-1"
        },
        "operation_identifier": {
          "index": 0
        },
        "type": "transfer"
      },
      {
        "account": {
          "address": "AGgdDesVBCBwNaVtEXX5LYaNckXv8qnC8d"
        },
        "amount": {
          "currency": {
            "symbol": "ONT",
            "decimals": 0,
            "metadata": {
              "contract": "0100000000000000000000000000000000000000"
            }
          },
          "value": "1"
        },
        "operation_identifier": {
          "index": 1
        },
        "related_operations": [
          {
            "index": 0
          }
        ],
        "type": "transfer"
      }
    ],
    "transaction_identifier": {
      "hash": "53eba2188f59fa4c8652fc13b5234acdd471ed1d03dcbb803ff9f16315e03ef3"
    }
  }
}
```

## Integrating using the Construction API

Please refer to the [dev document](https://docs.ont.io/ontology-node/node-deployment/rosetta-node#integrating-using-the-construction-api)
