FROM minio/mc:latest as mc


FROM drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /app
RUN export GO111MODULE=on \
  && cd /app \
  && CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o /usr/local/bin/boot boot.go


FROM alpine:3.12

RUN adduser \
	-s /bin/sh \
	-D \
	-h /home/git \
	git \
	git

COPY rootfs /
COPY --from=mc /usr/bin/mc /bin/mc
COPY --from=build /usr/local/bin/boot /usr/bin/boot

RUN apk add --update git sudo openssh-server coreutils tar xz jq bash \
  && mkdir -p /var/run/sshd \
  && rm -rf /etc/ssh/ssh_host* \
  && mkdir /apps \
  && passwd -u git \
  && chmod +x /bin/create_bucket /bin/normalize_storage /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
