# build stage
FROM golang:1.23 as build
LABEL maintainer="ea-trader"

WORKDIR /build

# copy source code
COPY . .

# fetch dependencies
RUN go mod download

# build application
RUN CGO_ENABLED=0 go build -o /app ./cmd

# runner stage
FROM alpine:3.10
# add certificates for HTTPS (if required)
RUN apk add --no-cache ca-certificates

# copy binary from build stage
COPY --from=build /app /usr/local/bin/app

# set binary as entry point
ENTRYPOINT ["/usr/local/bin/app"]