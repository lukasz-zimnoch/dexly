FROM golang:1.15.11-alpine3.13 AS build

ENV WORKDIR=/workdir \
	BIN_NAME=trading-service \
	BIN_PATH=/usr/local/bin

RUN apk add --update --no-cache \
	g++

RUN mkdir -p $WORKDIR

WORKDIR $WORKDIR

ADD . $WORKDIR

RUN go build -a -o $BIN_NAME .

FROM alpine:3.13

ENV WORKDIR=/workdir \
	BIN_NAME=trading-service \
	BIN_PATH=/usr/local/bin

COPY --from=build $WORKDIR/$BIN_NAME $BIN_PATH
COPY --from=build $WORKDIR/database/migrations /database/migrations

ENTRYPOINT ["trading-service"]