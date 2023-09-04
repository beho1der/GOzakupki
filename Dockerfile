FROM golang:1.13.6-alpine3.10 AS builder

RUN apk update && apk add openssh-client git
ADD . /go/src/GOzakupki
WORKDIR /go/src/GOzakupki
ENV GO111MODULE="on"
RUN go build 
RUN mkdir dist && mv GOzakupki dist/GOzakupki

FROM alpine:3.10.3
RUN apk --no-cache add ca-certificates bash && rm -rf /var/cache/apk/*
WORKDIR /app
COPY --from=builder /go/src/GOzakupki/dist ./
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["entrypoint.sh"]
