FROM golang:1.11.2

RUN mkdir /trezord-go
WORKDIR /trezord-go

RUN go get github.com/trezor/trezord-go
RUN go build github.com/trezor/trezord-go

CMD ["/trezord-go/trezord-go", "-e", "21324", "-u=false"]
EXPOSE 21325