# Build rosetta-node
FROM golang:1.13 AS build
WORKDIR /app
RUN git clone https://github.com/ontio/ontology-rosetta && \
  cd ontology-rosetta && \
  make rosetta-node

# Copy node binary from build
FROM golang:1.13
WORKDIR /app
COPY --from=build /app/ontology-rosetta/rosetta-node rosetta-node
COPY --from=build /app/ontology-rosetta/start.sh start.sh

EXPOSE 8080

# start.sh assumes there exists a volume mounted at /data that contains
# a rosetta-config.json file.
ENTRYPOINT ["/app/start.sh"]
