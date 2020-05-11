FROM ubuntu:18.04

WORKDIR /app

COPY ./ontology-rosetta /app/
COPY ./rosetta-config.json /app

EXPOSE 8080

#should have a volume mount for /data
CMD ["/app/ontology-rosetta", "--disable-log-file", "--data-dir", "/data/Chain"]
