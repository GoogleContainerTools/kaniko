package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// Hasher returns a hash function, used in snapshotting to determine if a file has changed
func Hasher() func(string) string {
	hasher := func(p string) string {
		h := md5.New()
		fi, err := os.Lstat(p)
		if err != nil {
			panic(err)
		}
		h.Write([]byte(fi.Mode().String()))
		h.Write([]byte(fi.ModTime().String()))

		if fi.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if _, err := io.Copy(h, f); err != nil {
				panic(err)
			}
		}

		return hex.EncodeToString(h.Sum(nil))
	}
	return hasher
}
