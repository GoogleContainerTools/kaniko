FROM rabbitmq@sha256:57b028a4bb9592ece3915e3e9cdbbaecb3eb82b753aaaf5250f8d25d81d318e2
# This base image has a volume declared at /var/lib/rabbitmq
# This is important because it should not exist in the child image.
COPY context/foo /usr/local/bin/
CMD ["script.sh"]
