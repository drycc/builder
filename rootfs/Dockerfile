FROM alpine:3.12

RUN adduser \
	-s /bin/sh \
	-D \
	-h /home/git \
	git \
	git

COPY . /

RUN wget https://dl.min.io/client/mc/release/linux-amd64/mc -O /bin/mc \
  && chmod +x /bin/mc \
  && apk add --update git sudo openssh-server coreutils tar xz jq bash \
  && mkdir -p /var/run/sshd \
  && rm -rf /etc/ssh/ssh_host* \
  && mkdir /apps \
  && passwd -u git \
  && chmod +x /bin/create_bucket /bin/normalize_storage /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
