#!/bin/bash

dir=fugo-invader-0.0

mkdir $dir

cp ~/go/bin/invader $dir
cp ~/go/bin/arena $dir
cp -a demo/invader/assets $dir
cp invader.apk $dir
cp RELEASE.md $dir

tar czf $dir.tar.gz $dir
