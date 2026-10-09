FROM golang:1.27.2-alpine@sha256:85dc1069ac644ea3c527b177303a406eb3358192816cd7f9e5848eb658851673 AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOWORK=off
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=development
RUN go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /gateway /gateway
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/gateway"]
