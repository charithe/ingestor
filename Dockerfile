FROM golang:1.11-alpine as builder
ARG SRC_DIR=/go/src/github.com/charithe/ingestor
RUN apk --no-cache add --update make git
RUN go get -u github.com/golang/dep/cmd/dep
ADD . $SRC_DIR
WORKDIR $SRC_DIR
RUN make container 

FROM gcr.io/distroless/base
COPY --from=builder /go/src/github.com/charithe/ingestor/ingestor /ingestor
ENTRYPOINT ["/ingestor"]
