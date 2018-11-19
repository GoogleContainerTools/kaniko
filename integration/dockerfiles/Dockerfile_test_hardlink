FROM composer@sha256:4598feb4b58b4370893a29cbc654afa9420b4debed1d574531514b78a24cd608 AS composer
FROM php@sha256:13813f20fec7ded7bf3a4305ea0ccd4df3cea900e263f7f86c3d5737f86669eb
COPY --from=composer /usr/bin/composer /usr/bin/composer

# make sure hardlink extracts correctly
FROM jboss/base-jdk@sha256:138591422fdab93a5844c13f6cbcc685631b37a16503675e9f340d2503617a41

FROM gcr.io/kaniko-test/hardlink-base:latest
RUN ls -al /usr/libexec/git-core/git /usr/bin/git /usr/libexec/git-core/git-diff
RUN stat /usr/bin/git
RUN stat /usr/libexec/git-core/git
RUN git --version > /git-version
