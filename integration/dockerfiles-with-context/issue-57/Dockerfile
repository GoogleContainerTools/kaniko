# Copyright 2020 Google, Inc. All rights reserved.
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

FROM busybox

RUN addgroup -g 1001 -S appgroup \
  && adduser -u 1005 -S al -G appgroup \
  && adduser -u 1004 -S bob -G appgroup

# Add local file with and without chown
ADD --chown=1005:appgroup a.txt /a.txt
RUN [ "$(ls -ln /a.txt | tr -s ' ' | cut -d' ' -f 3)" = 1005 ] && \
  [ "$(ls -ln /a.txt | tr -s ' ' | cut -d' ' -f 4)" = 1001 ]
ADD b.txt /b.txt

# Add remote file with and without chown
ADD --chown=bob:1001 https://raw.githubusercontent.com/GoogleContainerTools/kaniko/master/README.md /r1.txt
RUN [ "$(ls -ln /r1.txt | tr -s ' ' | cut -d' ' -f 3)" = 1004 ] && \
  [ "$(ls -ln /r1.txt | tr -s ' ' | cut -d' ' -f 4)" = 1001 ]
ADD https://raw.githubusercontent.com/GoogleContainerTools/kaniko/master/README.md /r2.txt
