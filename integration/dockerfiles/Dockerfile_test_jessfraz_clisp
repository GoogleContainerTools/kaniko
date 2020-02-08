FROM alpine:latest AS cl-k8s
RUN apk add --no-cache \
	git
RUN git clone https://github.com/brendandburns/cl-k8s.git /cl-k8s

FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	ca-certificates \
	clisp \
	wget \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

COPY .clisprc.lisp /home/user/.clisprc.lisp
COPY --from=cl-k8s /cl-k8s /home/user/quicklisp/local-projects/cl-k8s

# Install quicklisp
RUN wget -O /home/user/quicklisp.lisp https://beta.quicklisp.org/quicklisp.lisp

ENV HOME /home/user
RUN useradd --create-home --home-dir $HOME user \
	&& chown -R user:user $HOME

USER user

WORKDIR $HOME

# Install quicklisp
RUN clisp -x '(load "quicklisp.lisp") (quicklisp-quickstart:install)'

ENTRYPOINT [ "clisp" ]
