FROM golang AS builder

RUN echo $PWD && whoami

COPY . /go/app

WORKDIR /go/app

RUN make build

FROM alpine:3.18.3

# set labels for metadata
LABEL maintainer="Morven Cao<morvencao@gmail.com>" \
  name="image-rewrite" \
  description="A Kubernetes mutating webhook server that rewrites image urls to point to mirrors" \
  summary="A Kubernetes mutating webhook server that rewrites image urls to point to mirrors"

# set environment variables
ENV IMAGE_REWRITE=/usr/local/bin/image-rewrite \
  USER_UID=1001 \
  USER_NAME=image-rewrite

# install image-rewrite binary
COPY --from=builder /go/app/build/_output/bin/image-rewrite ${IMAGE_REWRITE}

# copy licenses
RUN mkdir /licenses
COPY --from=builder /go/app/LICENSE /licenses

# set entrypoint
ENTRYPOINT ["/usr/local/bin/image-rewrite"]

# switch to non-root user
USER ${USER_UID}
