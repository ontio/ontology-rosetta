#!/bin/bash

id=${NETWORK_ID:-1}

/app/rosetta-node --log-dir /data/Log --data-dir /data/Chain --server-config /data/server-config.json --networkid $id
