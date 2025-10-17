ARG LDFLAGS
ARG CODENAME

FROM registry.drycc.cc/drycc/go-dev:latest AS build
ADD . /workspace
RUN export GO111MODULE=on \
  && cd /workspace \
  && CGO_ENABLED=0 init-stack go build -ldflags "${LDFLAGS}" -o /usr/local/bin/boot boot.go

FROM registry.drycc.cc/drycc/base:${CODENAME}

ARG DRYCC_UID=1001 \
  DRYCC_GID=1001 \
  DRYCC_HOME_DIR=/workspace \
  RCLONE_VERSION="1.71.1" \
  JQ_VERSION="1.7.1"

RUN groupadd drycc --gid ${DRYCC_GID} \
  && useradd drycc -u ${DRYCC_UID} -g ${DRYCC_GID} -s /bin/bash -m -d ${DRYCC_HOME_DIR}

COPY rootfs/bin /bin/
COPY rootfs/etc/ssh /etc/ssh/
COPY rootfs/container-entrypoint.sh /container-entrypoint.sh
COPY --from=build /usr/local/bin/boot /usr/bin/boot

RUN install-packages git openssh-server coreutils xz-utils tar \
  && install-stack rclone $RCLONE_VERSION \
  && install-stack jq $JQ_VERSION \
  && mkdir -p /var/run/sshd \
  && rm -rf /etc/ssh/ssh_host* \
  && chmod +x /bin/create_bucket /container-entrypoint.sh

USER ${DRYCC_UID}
WORKDIR ${DRYCC_HOME_DIR}

ENTRYPOINT ["init-stack", "/container-entrypoint.sh"]
CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
