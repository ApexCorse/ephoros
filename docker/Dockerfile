FROM golang:alpine AS builder

ARG SERVICE

WORKDIR /opt/${SERVICE}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o bin/${SERVICE} cmd/${SERVICE}/main.go

FROM alpine

ARG SERVICE

WORKDIR /opt/${SERVICE}

COPY --from=builder /opt/${SERVICE}/bin/${SERVICE} bin/main

RUN chmod +x bin/main

CMD ["./bin/main"]
