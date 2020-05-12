FROM ubuntu:18.04

WORKDIR /app

COPY ./rosetta-node /app/
COPY ./start.sh /app

EXPOSE 8080

#should have a volume mount for /data
# append more rosetta-node args after image in docker run command, e.g. docker run ontology-rosetta:0.4 -- --network-id 2
ENTRYPOINT ["/app/start.sh"]
