FROM busybox

RUN adduser --disabled-password --gecos "" --uid 1000 user
RUN mkdir -p /home/user/foo
RUN chown -R user /home/user
RUN chmod 700 /home/user/foo
ADD https://raw.githubusercontent.com/GoogleContainerTools/kaniko/master/README.md /home/user/foo/README.md
RUN chown -R user /home/user
