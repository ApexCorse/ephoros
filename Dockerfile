FROM golang:alpine AS builder

ARG SERVICE

WORKDIR /opt/${SERVICE}

COPY ${SERVICE}/go.mod ${SERVICE}/go.sum ./
RUN go mod download

COPY ${SERVICE}/ .

RUN go build -o bin/${SERVICE} .

FROM alpine

ARG SERVICE

WORKDIR /opt/${SERVICE}

COPY --from=builder /opt/${SERVICE}/bin/${SERVICE} /usr/local/bin/main

RUN chmod +x /usr/local/bin/main

CMD ["/usr/local/bin/main"]
