# Make sure that whitelisting (specifically, filepath.SkipDir) works correctly, and that /var/test/testfile and
# /etc/test/testfile end up in the final image

FROM debian@sha256:38236c068c393272ad02db100e09cac36a5465149e2924a035ee60d6c60c38fe

RUN mkdir -p /var/test \
    && mkdir -p /etc/test \
    && touch /var/test/testfile \
    && touch /etc/test/testfile \
    && ls -lah /var/test \
    && ls -lah /etc/test;
