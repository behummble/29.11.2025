FROM golang:1.24.2-alpine as builder

ENV CGO_ENABLED 0
RUN apk update --no-cache && apk add --no-cache tzdata
WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download

COPY . .

RUN go build -o server ./app/main.go

FROM alpine
RUN apk update --no-cache && apk add --no-cache ca-certificates
COPY --from=builder /usr/share/zoneinfo/Europe/Moscow /usr/share/zoneinfo/Europe/Moscow
ENV TZ Europe/Moscow
WORKDIR /app
COPY --from=builder /build/server ./server
COPY --from=builder /build/config/config.yaml ./config/config.yaml

CMD [ "./server" ]