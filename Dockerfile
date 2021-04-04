# Builder
FROM golang:1.14-alpine3.11 AS builder

ARG dburl
ARG bSkey
ARG bPkey

WORKDIR /src
COPY . .
RUN go mod tidy

WORKDIR /src/cmd
RUN go build .

# Application
FROM golang:1.14-alpine3.11 AS app

COPY --from=builder /src/cmd/cmd /bin
WORKDIR /bin
CMD ["cmd"]


