# ontology-rosetta
Ontology node which follows Rosetta Blockchain Standard
## Build docker image

```sh
make
docker build -t ontology-rosetta:0.1 .
```

## Running docker image

```sh
docker run --name rosettatest -d -v /opt/data/Chain:/data/Chain -v /opt/data/rosetta-config.json:/data/rosetta-config.json -p 9090:8080 ontology-rosetta:0.1
```
