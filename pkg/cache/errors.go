/*
Copyright 2019 Google LLC

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

package cache

import "errors"

// IsAlreadyCached returns true if the supplied error is of the type AlreadyCachedErr
// otherwise it returns false.
func IsAlreadyCached(err error) bool {
	var e AlreadyCachedErr
	return errors.As(err, &e)
}

// AlreadyCachedErr is returned when the Docker image requested for caching is already
// present in the cache.
type AlreadyCachedErr struct {
	msg string
}

func (a AlreadyCachedErr) Error() string {
	return a.msg
}

// IsNotFound returns true if the supplied error is of the type NotFoundErr
// otherwise it returns false.
func IsNotFound(err error) bool {
	var e NotFoundErr
	return errors.As(err, &e)
}

// NotFoundErr is returned when the requested Docker image is not present in the cache.
type NotFoundErr struct {
	msg string
}

func (e NotFoundErr) Error() string {
	return e.msg
}

// IsExpired returns true if the supplied error is of the type ExpiredErr
// otherwise it returns false.
func IsExpired(err error) bool {
	var e ExpiredErr
	return errors.As(err, &e)
}

// ExpiredErr is returned when the requested Docker image is present in the cache, but is
// expired according to the supplied TTL.
type ExpiredErr struct {
	msg string
}

func (e ExpiredErr) Error() string {
	return e.msg
}
