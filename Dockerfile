FROM golang:1.12

WORKDIR /go/src/app
ADD hub_exporter.go ./


RUN go get -d -v ./...
RUN go install -v ./...
RUN go build hub_exporter.go

EXPOSE 9463

CMD ["./hub_exporter"]
