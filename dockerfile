FROM golang:1.17-alpine as build

WORKDIR /
COPY . .

RUN go build

FROM scratch

COPY --from=build /kube-event-responder /kube-event-responder

CMD ["/kube-event-responder", "run"]