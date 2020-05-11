# ontology-rosetta
Ontology node which follows Rosetta Blockchain Standard
## Build docker image

```sh
make
docker build -t ontology-rosetta:0.1 .
```

## Running docker image

```sh
docker run --name ontology-rosetta -d -v /data/ontology/:/data -p 9090:8080 ontology-rosetta:0.1
```
