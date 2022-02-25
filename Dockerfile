FROM docker.io/drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /app
RUN export GO111MODULE=on \
  && cd /app \
  && CGO_ENABLED=0 init-stack go build -ldflags "${LDFLAGS}" -o /usr/local/bin/boot boot.go


FROM docker.io/drycc/base:bullseye

RUN adduser --system \
   --shell /bin/sh \
   --home /home/git \
   --group \
   git

COPY rootfs/bin /bin/
COPY rootfs/etc/ssh /etc/ssh/
COPY rootfs/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=build /usr/local/bin/boot /usr/bin/boot

ENV MC_VERSION="RELEASE.2022-02-26T03-58-31Z" \
  JQ_VERSION="1.6"

RUN install-packages git openssh-server coreutils xz-utils tar \
  && install-stack mc $MC_VERSION \
  && install-stack jq $JQ_VERSION \
  && mkdir -p /var/run/sshd \
  && rm -rf /etc/ssh/ssh_host* \
  && mkdir /apps \
  && passwd -u git \
  && chmod +x /bin/create_bucket /bin/normalize_storage /docker-entrypoint.sh

ENTRYPOINT ["init-stack", "/docker-entrypoint.sh"]

CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
