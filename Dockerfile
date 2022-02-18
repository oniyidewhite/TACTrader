FROM golang:1.17 as build

MAINTAINER "ea-trader"

WORKDIR /build
COPY . .
RUN go mod download
RUN go mod tidy
RUN go mod vendor
RUN CGO_ENABLED=0 go build -o /app ./cmd

#Create runner
FROM alpine:3.10
COPY --from=build /app /
ENTRYPOINT ["/app"]
