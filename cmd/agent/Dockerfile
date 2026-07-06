FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /agent-service ./cmd/agent
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /agent-service /agent-service
EXPOSE 50056
ENTRYPOINT ["/agent-service"]
