ARG image
FROM ${image}
# First, make sure simple arg replacement works
ARG file
COPY $file /foo
# Check that setting a default value works
ARG file2=context/bar/bat
COPY $file2 /bat
# Check that overriding a default value works
ENV baz baz
ENV src file3
ARG ${src}=context/bar/${baz}
COPY $file3 /baz
# Check that setting an ENV will override the ARG
ENV file context/bar/bam/bat
COPY $file /env 

