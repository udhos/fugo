[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/fugo/blob/master/LICENSE)
[![Go Report Card - invader](https://goreportcard.com/badge/github.com/udhos/fugo/invader)](https://goreportcard.com/report/github.com/udhos/fugo/invader)

# fugo
fugo - fun with Go

QUICK START
===========

1) Install latest Go

    # There are many other ways, this is a quick recipe:
    git clone github.com/udhos/update-golang
    cd update-golang
    sudo ./update-golang.sh

2) Install Android NDK

    Install with Android Studio:
    https://developer.android.com/studio/install.html   

    # Then point the env var NDK to your ndk-bundle
    # For example:
    echo 'export NDK=$HOME/Android/Sdk/ndk-bundle' >> ~/.profile
    . ~/.profile

3) Install gomobile

    go get golang.org/x/mobile/cmd/gomobile
    gomobile init -ndk $NDK

4) Get fugo

    go get github.com/udhos/fugo

5) Build for desktop

    go install github.com/udhos/fugo/demo/invader

Hint: You can test the desktop version by running 'invader':

    $ invader slow ;# the slow parameter forces a very low frame rate

6) Build for Android

    gomobile build -target=android github.com/udhos/fugo/demo/invader

Hint: Use 'gomobile build -x' to see what the build is doing.

    $ gomobile build -x github.com/udhos/fugo/demo/invader

7) Push into Android device

    gomobile install github.com/udhos/fugo/demo/invader

--xx--

