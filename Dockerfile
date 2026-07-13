FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /opdeals-server ./cmd/server

FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /opdeals-server /usr/local/bin/opdeals-server
COPY --from=web /src/web/dist web/dist
VOLUME /app/data
EXPOSE 8080
ENTRYPOINT ["opdeals-server"]
CMD ["-web=web/dist", "-schedule"]
