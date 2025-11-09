FROM golang:1.22-alpine AS build
WORKDIR /app
COPY marketplace/service/go.mod ./go.mod
COPY marketplace/service/go.sum ./go.sum
COPY shared/coreauth /shared/coreauth
RUN apk add --no-cache git \
 && go env -w GOPROXY=https://proxy.golang.org,direct \
 && go mod download -x || true
COPY marketplace/service/ ./
RUN go mod tidy && go mod download -x || true
RUN mkdir -p /out \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/berjis-marketplace ./cmd/service

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/berjis-marketplace /berjis-marketplace
COPY --from=build /app/openapi /openapi
EXPOSE 8093
USER 65532:65532
ENTRYPOINT ["/berjis-marketplace"]
