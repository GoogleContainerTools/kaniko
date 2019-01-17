FROM busybox
RUN (while true; do sleep 10; dd if=/dev/zero of=file`date +%s`.txt count=16000 bs=256  > /dev/null 2>&1; done &); sleep 1
RUN echo "wait a second..." && sleep 2 && ls -lrat file*.txt || echo "test passed."
