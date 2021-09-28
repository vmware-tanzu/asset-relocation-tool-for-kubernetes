FROM golang:1.16 as mkcert
RUN cd /go && \
    GO111MODULE=off go get -u filippo.io/mkcert && \
    cd src/filippo.io/mkcert && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /go/bin/mkcert

FROM registry:2
COPY --from=mkcert /go/bin/mkcert /mkcert
COPY run-local-registry.sh /run-local-registry.sh
CMD /run-local-registry.sh
