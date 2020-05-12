#!/bin/bash

id=${NETWORK_ID:-1}

/app/rosetta-node --disable-log-file --data-dir /data/Chain --rosetta-config /data/rosetta-config.json --networkid $id
