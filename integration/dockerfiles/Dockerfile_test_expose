FROM gcr.io/google-appengine/debian9@sha256:1d6a9a6d106bd795098f60f4abb7083626354fa6735e81743c7f8cfca11259f0
EXPOSE 80
EXPOSE 81/udp
ENV protocol tcp
EXPOSE 82/$protocol
ENV port 83
EXPOSE $port/udp
EXPOSE $port/$protocol
