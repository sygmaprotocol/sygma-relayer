# Copyright 2020 ChainSafe Systems
# SPDX-License-Identifier: LGPL-3.0-only

FROM alpine:3.6 as alpine
RUN apk add -U --no-cache ca-certificates

FROM  golang:1.19-stretch AS builder
ADD . /src
WORKDIR /src
RUN cd /src && echo $(ls -1 /src)
RUN go mod download
RUN go build -ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore" -o /bridge .

# final stage
FROM debian:stretch-slim
COPY --from=builder /bridge ./
RUN chmod +x ./bridge
RUN mkdir -p /mount
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["./bridge"]
