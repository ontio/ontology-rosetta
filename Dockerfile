FROM golang:1.13 AS build

# Build node
WORKDIR /app
RUN git clone https://github.com/ontio/ontology-rosetta && \
  cd ontology-rosetta && \
  make rosetta-node

FROM golang:1.13

# Copy node binary from build
WORKDIR /app
COPY --from=build /app/ontology-rosetta/rosetta-node rosetta-node
COPY --from=build /app/ontology-rosetta/start.sh start.sh

EXPOSE 8080

#should have a volume mount for /data
# append more rosetta-node args after image in docker run command, e.g. docker run ontology-rosetta:0.4 -- --network-id 2
ENTRYPOINT ["/app/start.sh"]
