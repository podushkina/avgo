FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.work ./
COPY services/api/go.mod services/api/go.sum ./services/api/
RUN cd services/api && go mod download

COPY services/api/ ./services/api/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./services/api/cmd/server

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/server"]
