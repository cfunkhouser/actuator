FROM golang:1.16

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./cmd/actuate-exec/

EXPOSE 9942/tcp

CMD ["actuate-exec"]