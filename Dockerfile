FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev opus-dev pkgconfig

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /radio ./cmd/radio

# ---

FROM alpine:3.21

RUN apk add --no-cache ffmpeg opus ca-certificates

COPY --from=builder /radio /radio

EXPOSE 8080

CMD ["/radio"]
