FROM golang:latest

WORKDIR /app

COPY ./go.mod /app
COPY ./go.sum /app

RUN go mod download

COPY . /app

RUN go test ./...
RUN go build ./...
RUN go build .

CMD ["/app/csi-azure-fake-provider"]
