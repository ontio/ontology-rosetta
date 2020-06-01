#!/bin/bash

id=${NETWORK_ID:-1}

/app/rosetta-node --log-dir /data/Log --data-dir /data/Chain --rosetta-config /data/rosetta-config.json --networkid $id
