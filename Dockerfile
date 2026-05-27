FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/graphgate ./cmd/graphgate

FROM alpine:3.20
RUN adduser -D -H graphgate
USER graphgate
WORKDIR /work
COPY --from=build /out/graphgate /usr/local/bin/graphgate
ENTRYPOINT ["graphgate"]
