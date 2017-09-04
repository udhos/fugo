#!/bin/sh

step=0

msg() {
    step=$((step+1))
    echo >&2 $step. $*
}

get() {
    i=$1
    msg fetching $i
    go get $i
    msg fetching $i - done
}

pkg=github.com/udhos/fugo

check() {
    local sub=$1
    local full=$pkg/$sub

    msg fmt $sub
    gofmt -s -w $sub/*.go

    msg fix $sub
    go tool fix $sub/*.go

    msg vet $sub
    go tool vet $sub

    msg gosimple $sub
    hash gosimple && gosimple $sub/*.go

    msg golint $sub
    hash golint && golint $sub/*.go

    msg staticcheck $sub
    hash staticcheck && staticcheck $sub/*.go

    msg unused $sub
    hash unused && unused $sub/*.go

    msg test $full
    go test $full
}

build() {
    local sub=$1
    local full=$pkg/$sub

    check $sub

    msg desktop install $full
    go install $full
}

mobilebuild() {
    local sub=$1
    local full=$pkg/$sub

    build $sub

    msg android build $full
    gomobile build -target=android $full

    msg now use this command do push to android device:
    echo gomobile install $full
}

get honnef.co/go/tools/cmd/unused
get honnef.co/go/tools/cmd/gosimple
get honnef.co/go/tools/cmd/staticcheck
get github.com/golang/lint/golint
get github.com/udhos/pixfont
get github.com/udhos/goglmath
get golang.org/x/net/ipv4
get github.com/faiface/beep

check future
check msg
check trace
check unit
if [ "$1" != arena ]; then
	mobilebuild demo/square
	mobilebuild demo/triangle2
	mobilebuild demo/triangle
	mobilebuild demo/invader
fi
build demo/font 
build arena

