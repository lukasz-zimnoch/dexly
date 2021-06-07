FROM golang:1.15.11-alpine3.13 AS build

ENV WORKDIR=/workdir \
	BIN_NAME=trading

RUN apk add --update --no-cache \
	g++

RUN mkdir -p $WORKDIR

WORKDIR $WORKDIR

ADD . $WORKDIR

RUN go build -a -o ./bin/$BIN_NAME ./cmd/$BIN_NAME

FROM alpine:3.13

ENV WORKDIR=/workdir \
	BIN_NAME=trading

COPY --from=build $WORKDIR/bin/$BIN_NAME /usr/local/bin
COPY --from=build $WORKDIR/database/migrations /database/migrations

ENTRYPOINT ["trading"]