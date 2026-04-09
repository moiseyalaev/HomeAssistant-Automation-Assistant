FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ha-bridge .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/ha-bridge .
EXPOSE 8080
CMD ["./ha-bridge"]
