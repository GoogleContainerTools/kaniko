#!/usr/local/bin/bash

mkdir /workdir

i=1
while [ $i -le $1 ]
do
  cat context.txt >  /workdir/somefile$i
  i=$(( $i + 1 ))
done
