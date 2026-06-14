FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_TARGET
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./${APP_TARGET}

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/app /app/app

USER app

ENTRYPOINT ["/app/app"]
