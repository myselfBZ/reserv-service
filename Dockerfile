FROM golang:1.26.0 as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api cmd/api/*.go

FROM scratch
WORKDIR /app
COPY --from=builder /api ./api

EXPOSE 8080
CMD ["./api"]
