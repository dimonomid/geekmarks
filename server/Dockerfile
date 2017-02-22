FROM golang:1.7

RUN apt-get update \
 && apt-get install -y --no-install-recommends netcat \
 && rm -rf /var/lib/apt/lists/*

COPY scripts/start.sh /start.sh
COPY geekmarks-server /geekmarks-server

CMD ["./start.sh", "postgres", "/geekmarks-server", "--logtostderr --v=5 --geekmarks.port=4000 --geekmarks.postgres.url=postgres://geekmarks:geekmarks@postgres:5432/geekmarks?sslmode=disable --google_oauth_creds_file=/data/google_oauth_creds.yaml"]

EXPOSE 4000