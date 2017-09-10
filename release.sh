#!/bin/bash

version=`grep 'Version =' version/version.go | awk '{ print $3 }'`
v=`eval echo $version`

dir=fugo-invader-$v

mkdir $dir

cp ~/go/bin/invader $dir
cp ~/go/bin/arena $dir
cp -a demo/invader/assets $dir
cp invader.apk $dir
cp RELEASE.md $dir
cp LICENSE $dir

tar czf $dir.tar.gz $dir

rm -r $dir

echo $dir.tar.gz
