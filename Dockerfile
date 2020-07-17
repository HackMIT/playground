FROM golang:alpine as builder
WORKDIR /workspace

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o ./app .

FROM alpine
WORKDIR /workspace

COPY config/base.json .
COPY --from=builder /workspace/app .
