package sysx

<<<<<<< HEAD
import "golang.org/x/sys/unix"
=======
import "syscall"

// These functions will be generated by generate.sh
//    $ GOOS=linux GOARCH=386 ./generate.sh xattr
//    $ GOOS=linux GOARCH=amd64 ./generate.sh xattr
//    $ GOOS=linux GOARCH=arm ./generate.sh xattr
//    $ GOOS=linux GOARCH=arm64 ./generate.sh xattr
//    $ GOOS=linux GOARCH=ppc64 ./generate.sh xattr
//    $ GOOS=linux GOARCH=ppc64le ./generate.sh xattr
//    $ GOOS=linux GOARCH=s390x ./generate.sh xattr
>>>>>>> WIP: set the docker default seccomp profile in the executor process.

// Listxattr calls syscall listxattr and reads all content
// and returns a string array
func Listxattr(path string) ([]string, error) {
<<<<<<< HEAD
	return listxattrAll(path, unix.Listxattr)
=======
	return listxattrAll(path, syscall.Listxattr)
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
}

// Removexattr calls syscall removexattr
func Removexattr(path string, attr string) (err error) {
<<<<<<< HEAD
	return unix.Removexattr(path, attr)
=======
	return syscall.Removexattr(path, attr)
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
}

// Setxattr calls syscall setxattr
func Setxattr(path string, attr string, data []byte, flags int) (err error) {
<<<<<<< HEAD
	return unix.Setxattr(path, attr, data, flags)
=======
	return syscall.Setxattr(path, attr, data, flags)
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
}

// Getxattr calls syscall getxattr
func Getxattr(path, attr string) ([]byte, error) {
<<<<<<< HEAD
	return getxattrAll(path, attr, unix.Getxattr)
}

// LListxattr lists xattrs, not following symlinks
func LListxattr(path string) ([]string, error) {
	return listxattrAll(path, unix.Llistxattr)
}

// LRemovexattr removes an xattr, not following symlinks
func LRemovexattr(path string, attr string) (err error) {
	return unix.Lremovexattr(path, attr)
}

// LSetxattr sets an xattr, not following symlinks
func LSetxattr(path string, attr string, data []byte, flags int) (err error) {
	return unix.Lsetxattr(path, attr, data, flags)
}

// LGetxattr gets an xattr, not following symlinks
func LGetxattr(path, attr string) ([]byte, error) {
	return getxattrAll(path, attr, unix.Lgetxattr)
=======
	return getxattrAll(path, attr, syscall.Getxattr)
}

//sys llistxattr(path string, dest []byte) (sz int, err error)

// LListxattr lists xattrs, not following symlinks
func LListxattr(path string) ([]string, error) {
	return listxattrAll(path, llistxattr)
}

//sys lremovexattr(path string, attr string) (err error)

// LRemovexattr removes an xattr, not following symlinks
func LRemovexattr(path string, attr string) (err error) {
	return lremovexattr(path, attr)
}

//sys lsetxattr(path string, attr string, data []byte, flags int) (err error)

// LSetxattr sets an xattr, not following symlinks
func LSetxattr(path string, attr string, data []byte, flags int) (err error) {
	return lsetxattr(path, attr, data, flags)
}

//sys lgetxattr(path string, attr string, dest []byte) (sz int, err error)

// LGetxattr gets an xattr, not following symlinks
func LGetxattr(path, attr string) ([]byte, error) {
	return getxattrAll(path, attr, lgetxattr)
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
}
