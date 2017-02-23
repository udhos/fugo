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

get github.com/gopherjs/gopherjs
get github.com/gopherjs/gopherjs/js
get github.com/gopherjs/webgl
get github.com/udhos/goglmath
get github.com/udhos/gwob
#get honnef.co/go/simple/cmd/gosimple
#get honnef.co/go/tools/cmd/staticcheck

src=`find . -type f | egrep '\.go$'`

msg fmt
gofmt -s -w $src
msg fix
go tool fix $src
msg vet
go tool vet .

msg install
pkg=github.com/udhos/fugo
# 1: gopherjs_bin=
if [ -z "$gopherjs_bin" ]; then
    gopherjs_bin=`which gopherjs` ;# 2: $PATH
fi
if [ ! -x "$gopherjs_bin" ]; then
    gopherjs_bin=$GOPATH/bin/gopherjs ;# 3: $GOPATH
fi
if [ ! -x "$gopherjs_bin" ]; then
    gopherjs_bin=$HOME/go/bin/gopherjs ;# 4: $HOME/go
fi
[ -x "$gopherjs_bin" ] || die "could not find gopherjs: gopherjs_bin=[$gopherjs_bin]"
$gopherjs_bin install $pkg/examples/fugo-simple-js
[ -w "$GOPATH/bin" ] && cp examples/fugo-simple-js/fugo-simple-js.html $GOPATH/bin
[ -w "$HOME/go/bin" ] && cp examples/fugo-simple-js/fugo-simple-js.html $HOME/go/bin

# go get honnef.co/go/simple/cmd/gosimple
s=$GOPATH/bin/gosimple
simple() {
    msg simple - this is slow, please standby
    # gosimple cant handle source files from multiple packages
    $s examples/fugo-simple-js/*.go
}
[ -x "$s" ] && simple

# go get github.com/golang/lint/golint
l=$GOPATH/bin/golint
lint() {
    msg lint
    # golint cant handle source files from multiple packages
    $l examples/fugo-simple-js/*.go
}
[ -x "$l" ] && lint

# go get honnef.co/go/tools/cmd/staticcheck
sc=$GOPATH/bin/staticcheck
static() {
    msg staticcheck - this is slow, please standby
    # staticcheck cant handle source files from multiple packages
    $sc examples/fugo-simple-js/*.go
}
[ -x "$sc" ] && static

msg test
go test $pkg/examples/fugo-simple-js
