/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutil

const (
	// Dockerfile is used for unit testing
	Dockerfile = `
	FROM scratch
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2

	FROM second
	RUN xxx
	
	FROM scratch
	COPY --from=second /hi2 /hi3

	FROM ubuntu:16.04 AS base
	ENV DEBIAN_FRONTEND noninteractive
	ENV LC_ALL C.UTF-8

	FROM base AS development
	ENV PS1 " 🐳 \[\033[1;36m\]\W\[\033[0;35m\] # \[\033[0m\]"

	FROM development AS test
	ENV ORG_ENV UnitTest

	FROM base AS production
	COPY . /code
	`
)
