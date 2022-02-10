ARG OTHER="fix"
FROM alpine:latest as base
ARG NAME="base"
RUN echo "base:: $NAME" >> A.txt

FROM base AS base-dev
ARG NAME="$NAME-dev"
RUN echo "dev:: $NAME" >> B.txt

FROM base-dev as base-custom
ARG NAME
RUN echo "custom:: $NAME" >> C.txt
RUN echo "custom:: $OTHER" >> C.txt

FROM base-custom as base-custom2
ARG OTHER
RUN echo "custom:: $NAME" >> D.txt
RUN echo "custom:: $OTHER" >> D.txt