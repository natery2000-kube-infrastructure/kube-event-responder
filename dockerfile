FROM golang:1.17.1-alpine3.14

RUN GOPROXY=direct
RUN go install github.com/natery2000-kube-infrastructure/kube-event-responder@latest

CMD ["kube-event-responder", "run"]