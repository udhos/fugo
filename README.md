# fugo
fugo - fun with Go

QUICK START
===========

1\. Install latest Go

    # There are many other ways, this is a quick recipe:
    git clone github.com/udhos/update-golang
    cd update-golang
    sudo ./update-golang.sh

2\. Add ~/go/bin to your PATH

    echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.profile
    . ~/.profile

3\. Install Android NDK

    # Then point the env var NDK to your install
    # For example:
    echo 'export NDK=$HOME/Android/Sdk/ndk-bundle' >> ~/.profile
    . ~/.profile

4\. Install gomobile

    go get go get golang.org/x/mobile/cmd/gomobile
    gomobile init -ndk $NDK

5\. Get fugo

    go get github.com/udhos/fugo

6\. Build for desktop

    go install github.com/udhos/fugo/invader

7\. Build for Android

    gomobile build -target=android github.com/udhos/fugo/invader

8\. Push into Android device

    gomobile install github.com/udhos/fugo/invader

