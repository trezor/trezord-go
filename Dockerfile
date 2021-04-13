FROM golang:1.11.2

RUN mkdir /onekey-go
WORKDIR /onekey-go
COPY ./scripts/run_in_docker.sh /onekey-go

RUN apt-get update
RUN apt-get install -y redir

RUN go get github.com/OneKeyHQ/onekey-go
RUN go build github.com/OneKeyHQ/onekey-go

ENTRYPOINT '/onekey-go/run_in_docker.sh'
EXPOSE 11325