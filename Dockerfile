FROM instrumentisto/dep:0.4.1 AS builder
ENV GOPATH     /go
ENV SRC_PATH   $GOPATH/src/github.com/ipfs/kubernetes-ipfs
WORKDIR $SRC_PATH

COPY . $SRC_PATH
RUN \
  dep init -v \
  && dep ensure -v \
  && go build

# ===

FROM busybox:1-glibc
ENV GOPATH     /go

COPY --from=builder \
  /go/src/github.com/ipfs/kubernetes-ipfs /code/

