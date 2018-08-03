FROM alpine@sha256:5ce5f501c457015c4b91f91a15ac69157d9b06f1a75cf9107bf2b62e0843983a
COPY context/foo foo
COPY context/foo /foodir/
COPY context/bar/b* bar/
COPY context/fo? /foo2
COPY context/bar/doesnotexist* context/foo hello
COPY ./context/empty /empty
COPY ./ dir/
COPY . newdir
COPY context/bar /baz/
COPY ["context/foo", "/tmp/foo" ]
COPY context/q* /qux/
COPY context/foo context/bar/ba? /test/
COPY context/arr[[]0].txt /mydir/
COPY context/bar/bat .

ENV contextenv ./context
COPY ${contextenv}/foo /tmp/foo2
COPY $contextenv/foo /tmp/foo3
COPY $contextenv/* /tmp/${contextenv}/
