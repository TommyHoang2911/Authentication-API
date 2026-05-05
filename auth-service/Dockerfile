FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-api ./cmd/server

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/auth-api .

COPY --from=builder /app/db/migrations ./db/migrations

COPY --from=builder /app/templates ./templates

COPY --from=builder /app/static ./static

EXPOSE 8080

CMD [ "./auth-api" ]
