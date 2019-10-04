# Test to make sure the cache works with special file permissions properly.
# If the image is built twice, directory foo should have the sticky bit,
# and file bar should have the setuid and setgid bits.

FROM busybox

RUN mkdir foo && chmod +t foo
RUN touch bar && chmod u+s,g+s bar
