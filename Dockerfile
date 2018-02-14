# Builds the static image with the Go builder executable
FROM scratch
ADD main /work-dir/
ADD files/ca-certificates.crt /etc/ssl/certs/
ADD files/policy.json /etc/containers/
ADD files/docker-credential-gcr_linux_amd64-1.4.1.tar.gz /usr/local/bin/
ADD files/config.json /root/.docker/