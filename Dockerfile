FROM golang:alpine

WORKDIR /build
COPY go.mod go.sum app.go /build/
RUN go mod download
RUN go build -o /app app.go

FROM alpine
COPY --from=0 /app /app
CMD /app
