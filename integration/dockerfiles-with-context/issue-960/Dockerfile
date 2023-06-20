# Copyright 2023 Google, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM alpine:3.14 as rootfs

FROM alpine:3.14

RUN mkdir -p /sysroot
COPY --from=rootfs / /sysroot/

# Workaround: we must remove some files to pass the integration test.
# Unlike Docker, kaniko has no access to the original layer data from the
# building context and can't preserve them in their original form.
RUN rm -f \
      /sysroot/etc/hostname \
      /sysroot/etc/hosts \
      /sysroot/etc/mtab \
      /sysroot/etc/nsswitch.conf

# Additional check for preserved dirs. They must persist in image but be empty.
RUN printf "%s\n" \
      "/sysroot/dev/:" \
      "" \
      "/sysroot/sys/:" \
      > /tmp/expected \
 && ls -1 /sysroot/dev/ /sysroot/sys/ \
      > /tmp/got \
 && diff -u /tmp/got /tmp/expected