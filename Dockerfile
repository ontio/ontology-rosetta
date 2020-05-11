FROM ubuntu:18.04

WORKDIR /app

COPY ./rosetta-node /app/

EXPOSE 8080

#should have a volume mount for /data
CMD ["/app/rosetta-node", "--disable-log-file", "--data-dir", "/data/Chain", "--rosetta-config", "/data/rosetta-config.json"]
