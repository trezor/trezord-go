FROM golang:1.22.1

RUN mkdir /trezord-go
WORKDIR /trezord-go
COPY . /trezord-go

RUN apt-get update
RUN apt-get install -y redir

RUN go build .

ENTRYPOINT '/trezord-go/scripts/run_in_docker.sh'
EXPOSE 11325
