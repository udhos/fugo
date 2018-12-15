#!/bin/sh

step=0

msg() {
    step=$((step+1))
    echo >&2 $step. $*
}

check() {
    local sub=$1

    msg fmt $sub
    gofmt -s -w $sub/*.go

    msg fix $sub
    go tool fix $sub/*.go

    msg vet $sub
    go tool vet $sub

    #msg gosimple $sub
    #hash gosimple && gosimple $sub/*.go

    msg golint $sub
    hash golint && golint $sub/*.go

    #msg staticcheck $sub
    #hash staticcheck && staticcheck $sub/*.go

    #msg unused $sub
    #hash unused && unused $sub/*.go

    msg test $sub
    go test $sub
}

build() {
    local sub=$1

    check $sub

    msg desktop install $sub
    go install $sub
}

mobilebuild() {
    local sub=$1

    build $sub

    msg android build $sub
    gomobile build -target=android $sub

    msg now use this command do push to android device:
    echo gomobile install $sub
}

check ./future
check ./msg
check ./trace
check ./unit
check ./version

if [ "$1" != arena ]; then
	mobilebuild ./demo/square
	mobilebuild ./demo/triangle2
	mobilebuild ./demo/triangle
	mobilebuild ./demo/invader
fi

build ./demo/font 
build ./arena

