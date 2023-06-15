# Copyright 2020 ChainSafe Systems
# SPDX-License-Identifier: LGPL-3.0-only

FROM  golang:1.19 AS builder
ADD . /src
WORKDIR /src
RUN cd /src && echo $(ls -1 /src)
RUN go mod download
RUN go build -ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore" -o /bridge ./example/

# final stage
FROM debian:stable-slim
COPY --from=builder /bridge ./
RUN chmod +x ./bridge

CMD ["./bridge"]
