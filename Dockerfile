FROM golang:alpine3.20 as builder

WORKDIR /build
COPY . .
#RUN apk update && apk add --no-cache make protobuf-dev
RUN apk update && apk add --no-cache make

RUN make

FROM alpine:3.20

WORKDIR /
COPY --from=builder /build/server server

ENTRYPOINT ["/server"]
