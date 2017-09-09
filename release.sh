#!/bin/bash

dir=fugo-invader-0.1

mkdir $dir

cp ~/go/bin/invader $dir
cp ~/go/bin/arena $dir
cp -a demo/invader/assets $dir
cp invader.apk $dir
cp RELEASE.md $dir

tar czf $dir.tar.gz $dir

rm -r $dir

echo $dir.tar.gz
