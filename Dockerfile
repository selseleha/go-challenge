FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /estimator ./cmd/estimator

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /estimator /usr/local/bin/estimator
EXPOSE 8080 9090
ENTRYPOINT ["estimator"]
