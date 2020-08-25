FROM minio/mc:RELEASE.2020-07-17T02-52-20Z as mc

FROM alpine:3.12

RUN adduser \
	-s /bin/sh \
	-D \
	-h /home/git \
	git \
	git

COPY . /

COPY --from=mc /usr/bin/mc /usr/bin/mc

RUN  apk add --update git sudo openssh-server coreutils tar xz jq bash\
    && mkdir -p /var/run/sshd  \
    && rm -rf /etc/ssh/ssh_host*  \
	&& mkdir /apps  \
	&& passwd -u git \
    && chmod +x /bin/create_bucket /bin/normalize_storage /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
