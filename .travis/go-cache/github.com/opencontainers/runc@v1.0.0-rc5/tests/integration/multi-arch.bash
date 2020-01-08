#!/bin/bash
get_busybox(){
	case $(go env GOARCH) in
	arm64)
		echo 'https://github.com/docker-library/busybox/raw/23fbd9c43e0f4bec7605091bfba23db278c367ac/glibc/busybox.tar.xz'
	;;
	*)
		echo 'https://github.com/docker-library/busybox/raw/a0558a9006ce0dd6f6ec5d56cfd3f32ebeeb815f/glibc/busybox.tar.xz'
	;;
	esac
}

get_hello(){
	case $(go env GOARCH) in
        arm64)
                echo 'hello-world-aarch64.tar'
        ;;
        *)
                echo 'hello-world.tar'
        ;;
        esac
}
