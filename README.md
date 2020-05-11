# ontology-rosetta
Ontology node which follows Rosetta Blockchain Standard
## Build docker image

```sh
make
docker build -t rosetta:0.1 .
```

## Running docker image

```sh
docker run --name rosetta -d -v /data/ontology/:/data -p 9090:8080 rosetta:0.1
```
