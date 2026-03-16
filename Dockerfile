FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/gofer ./cmd/gofer

FROM alpine
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/gofer /bin/gofer
ENTRYPOINT ["/bin/gofer"]