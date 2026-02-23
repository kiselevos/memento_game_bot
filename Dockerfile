FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG APP_PATH=./cmd/main
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app ${APP_PATH}


FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/app /app/app

ENTRYPOINT ["/app/app"]