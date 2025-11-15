FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o assigner ./cmd/assigner

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/assigner .
COPY --from=builder /app/prod.env ./prod.env
CMD ["./assigner"]