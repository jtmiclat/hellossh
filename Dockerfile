FROM golang as builder

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/hellossh

FROM ubuntu:20.04
COPY --from=builder /go/bin/hellossh /

ADD ./assets /app/assets
ADD ./tmp /app/tmp
ENV ID_RSA_FILE="/app/tmp/id_rsa"
WORKDIR /app/assets

CMD ["/hellossh"]