FROM gcr.io/google-appengine/debian9@sha256:6b3aa04751aa2ac3b0c7be4ee71148b66d693ad212ce6d3244bd2a2a147f314a
COPY context/foo foo
WORKDIR /test
# Test that this will be appended on to the previous command, to create /test/workdir
WORKDIR workdir 
COPY context/foo ./currentfoo
# Test that the RUN command will happen in the correct directory
RUN cp currentfoo newfoo
WORKDIR /new/dir
ENV dir /another/new/dir
WORKDIR $dir/newdir
WORKDIR $dir/$doesntexist
WORKDIR /

# Test with ARG
ARG workdir
WORKDIR $workdir
