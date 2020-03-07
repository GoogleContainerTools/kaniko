FROM busybox as t
RUN mkdir temp
RUN echo "hello" > temp/target
RUN ln -s target temp/link
## Relative link with paths
RUN mkdir workdir && cd workdir && ln -s ../temp/target relative_link

FROM scratch
COPY --from=t temp/ dest/
COPY --from=t /workdir/relative_link /workdirAnother/