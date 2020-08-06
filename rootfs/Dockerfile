FROM alpine:3.12

RUN adduser \
	-s /bin/sh \
	-D \
	-h /home/git \
	git \
	git

RUN  sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --update git sudo openssh-server coreutils tar xz \
    && mkdir -p /var/run/sshd  \
    && rm -rf /etc/ssh/ssh_host*  \
	&& mkdir /apps  \
	&& passwd -u git

COPY . /

CMD ["/usr/bin/boot", "server"]

EXPOSE 2223
EXPOSE 3000
