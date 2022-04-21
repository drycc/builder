FROM registry.drycc.cc/drycc/go-dev:latest AS build
ARG LDFLAGS
ADD . /workspace
RUN export GO111MODULE=on \
  && cd /workspace \
  && CGO_ENABLED=0 init-stack go build -ldflags "${LDFLAGS}" -o /usr/local/bin/boot boot.go


FROM registry.drycc.cc/drycc/base:bullseye

ENV DRYCC_UID=1001 \
  DRYCC_GID=1001 \
  DRYCC_HOME_DIR=/workspace \
  MC_VERSION="2022.04.01.23.44.48" \
  JQ_VERSION="1.6"

RUN groupadd drycc --gid ${DRYCC_GID} \
  && useradd drycc -u ${DRYCC_UID} -g ${DRYCC_GID} -s /bin/bash -m -d ${DRYCC_HOME_DIR}

COPY rootfs/bin /bin/
COPY rootfs/etc/ssh /etc/ssh/
COPY rootfs/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=build /usr/local/bin/boot /usr/bin/boot

RUN install-packages git openssh-server coreutils xz-utils tar \
  && install-stack mc $MC_VERSION \
  && install-stack jq $JQ_VERSION \
  && mkdir -p /var/run/sshd \
  && rm -rf /etc/ssh/ssh_host* \
  && chmod +x /bin/create_bucket /docker-entrypoint.sh

USER ${DRYCC_UID}
WORKDIR ${DRYCC_HOME_DIR}

ENTRYPOINT ["init-stack", "/docker-entrypoint.sh"]
CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
