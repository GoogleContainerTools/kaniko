# Copyright 2021 Google, Inc. All rights reserved.
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

FROM alpine:3.11 as builder

RUN mkdir -p /myapp/somedir \
 && touch /myapp/somedir/somefile \
 && chown 123:123 /myapp/somedir \
 && chown 321:321 /myapp/somedir/somefile

FROM alpine:3.11
COPY --from=builder /myapp /myapp
RUN printf "%s\n" \
      "0 0 /myapp/" \
      "123 123 /myapp/somedir" \
      "321 321 /myapp/somedir/somefile" \
      > /tmp/expected \
 && stat -c "%u %g %n" \
      /myapp/ \
      /myapp/somedir \
      /myapp/somedir/somefile \
      > /tmp/got \
 && diff -u /tmp/got /tmp/expected
