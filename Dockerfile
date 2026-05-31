FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

ARG SERVICE
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/trackflow ./cmd/${SERVICE}

FROM alpine:3.20

RUN apk add --no-cache \
    chromium \
    font-noto \
    font-liberation \
    ttf-dejavu

RUN adduser -D -H -u 10001 trackflow
USER trackflow

COPY --from=build /out/trackflow /usr/local/bin/trackflow

ENTRYPOINT ["/usr/local/bin/trackflow"]
