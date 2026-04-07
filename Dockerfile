FROM golang:1.21-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/shukisha-api ./cmd/api
RUN go build -trimpath -ldflags="-s -w" -o /out/shukisha-worker ./cmd/worker


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=build /out/shukisha-api /app/shukisha-api
COPY --from=build /out/shukisha-worker /app/shukisha-worker
COPY docs/openapi.yaml /app/docs/openapi.yaml

EXPOSE 8080

