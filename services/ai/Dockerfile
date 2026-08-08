FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.work go.work.sum ./
COPY pkg/httpx/go.mod pkg/httpx/
COPY services/api/go.mod services/api/go.sum ./services/api/
COPY services/ai/go.mod services/ai/go.sum ./services/ai/

RUN cd services/ai && go mod download

COPY pkg/ ./pkg/
COPY services/ai/ ./services/ai/

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./services/ai/cmd/server

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/server"]