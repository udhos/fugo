#!/bin/bash

version=`grep 'const Version =' version/version.go | awk '{ print $4 }'`
v=`eval echo $version`

gen_release_md() {
	local i=$1
	cat <<__EOF__
# fugo invader release

Home: https://github.com/udhos/fugo

    $ tar xf $i.tar.gz
    $ cd $i

Tarball contents:

- arena: Server binary for desktop Linux
- invader: Client binary for desktop Linux
- invader.apk: Client package for Android
- assets: Game assets, required to run the Arena server

Run the server:

    $ arena

Run the desktop Linux client:

    $ invader
__EOF__
}

dir=fugo-invader-$v
mkdir $dir

cp ~/go/bin/invader $dir
cp ~/go/bin/arena $dir
cp -a demo/invader/assets $dir
cp invader.apk $dir
cp LICENSE $dir

gen_release_md $dir > $dir/RELEASE.md

tar czf $dir.tar.gz $dir
rm -r $dir
echo $dir.tar.gz

