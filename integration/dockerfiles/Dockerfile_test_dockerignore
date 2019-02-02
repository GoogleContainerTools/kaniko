# This dockerfile makes sure the .dockerignore is working
# If so then ignore/foo should copy to /foo
# If not, then this image won't build because it will attempt to copy three files to /foo, which is a file not a directory
FROM scratch 
COPY ignore/* /foo
