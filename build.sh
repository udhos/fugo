#!/bin/sh

step=0

msg() {
    step=$((step+1))
    echo >&2 $step. $*
}

die() {
    msg $*
    exit 1
}

get() {
    i=$1
    msg fetching $i
    go get $i
    msg fetching $i - done
}

#!/bin/bash

me=`basename $0`
msg() {
    echo >&2 $me: $*
}

pkg=github.com/udhos/fugo

build() {

    local sub=$1
    local full=$pkg/$sub

    msg fmt $sub
    gofmt -s -w $sub/*.go

    msg fix $sub
    go tool fix $sub/*.go

    msg vet $sub
    go tool vet $sub

    msg gosimple $sub
    [ -x ~go/bin/gosimple ] && ~go/bin/gosimple $sub/*.go

    msg golint $sub
    [ -x ~go/bin/golint ] && ~go/bin/golint $sub/*.go

    msg staticcheck $sub
    [ -x ~go/bin/staticcheck ] && ~go/bin/staticcheck $sub/*.go

    msg test $full
    go test $full

    msg desktop install $full
    go install $full

    msg android build $full
    gomobile build -target=android $full

    msg now use this command do push to android device:
    echo gomobile install $full
}

get honnef.co/go/simple/cmd/gosimple
get honnef.co/go/tools/cmd/staticcheck
get github.com/golang/lint/golint

build demo/triangle
build demo/invader

