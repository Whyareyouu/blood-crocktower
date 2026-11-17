FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 blood && adduser -S -u 10001 -G blood blood
WORKDIR /app
COPY --from=build /out/api /out/migrate ./
USER blood
EXPOSE 8080
CMD ["/app/api"]
