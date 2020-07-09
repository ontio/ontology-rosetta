module github.com/ontio/ontology-rosetta

go 1.13

require (
	github.com/coinbase/rosetta-sdk-go v0.2.0
	github.com/ethereum/go-ethereum v1.9.15
	github.com/ontio/ontology v1.11.0
	github.com/ontio/ontology-crypto v1.0.9
	github.com/ontio/ontology-eventbus v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/urfave/cli v1.22.4
)

replace github.com/coinbase/rosetta-sdk-go v0.2.0 => ../../coinbase/rosetta-sdk-go
