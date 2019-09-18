[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/fugo/blob/master/LICENSE)
[![Go Report Card - invader](https://goreportcard.com/badge/github.com/udhos/fugo/invader)](https://goreportcard.com/report/github.com/udhos/fugo/invader)

# fugo
fugo - fun with Go. gomobile OpenGL game

Table of Contents
=================

  * [QUICK START](#quick-start)
  * [Requirements](#requirements)
  * [Building the INVADER application](#building-the-invader-application)
  * [Building the ARENA server](#building-the-arena-server)
  * [How does the INVADER application locate the ARENA server?](#how-does-the-invader-application-locate-the-arena-server)
  * [INVADER runtime flags](#invader-runtime-flags)
  * [KNOWN ISSUES](#known-issues)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

## QUICK START

Recipe:

    go get github.com/udhos/fugo
    cd ~/go/src/github.com/udhos/fugo
    ./build.sh

## Requirements

1\. Install latest Go

There are many other ways, this is a quick recipe:

    git clone github.com/udhos/update-golang
    cd update-golang
    sudo ./update-golang.sh

2\. Install Android NDK

Download Android Studio - https://developer.android.com/studio

Unzip Android Studio:

    $ tar xf /tmp/android-studio-ide-191.5791312-linux.tar.gz

Run Android Studio:

    $ ~/android-studio/bin/studio.sh &

Select: Configure -> SDK Manager -> SDK Tools -> NDK

Click the Apply button.

Define SDK env vars. For example:

    export ANDROID_HOME=~/Android/Sdk
    export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/20.0.5594570 ;# watch out the version

3\. Install gomobile

Recipe:

    go get golang.org/x/mobile/cmd/gomobile
    gomobile version
    #gomobile init -ndk $NDK ;# no longer used?

4\. Install OpenGL dev libs

On Ubuntu you will need these:

    sudo apt install libegl1-mesa-dev libgles2-mesa-dev libx11-dev

5\. Install alsa sound dev libs

On Ubuntu you will need this:

    sudo apt install libasound2-dev

6\. Get fugo

Recipe:

    go get github.com/udhos/fugo

## Building the INVADER application

7\. Build for desktop

Recipe:

    go install -tags gldebug github.com/udhos/fugo/demo/invader

Hint: You can test the desktop version by running 'invader':

    $ (cd demo/invader && invader slow)

The parameter 'slow' sets a very low frame rate, useful for test/debugging.
If you want smooth rendering, remove the parameter 'slow'.

The subshell is used to temporarily enter the demo/invader dir in order to load assets from demo/invader/assets.

8\. Build for Android

Recipe:

    gomobile build -target=android github.com/udhos/fugo/demo/invader

Hint: Use 'gomobile build -x' to see what the build is doing.

    $ gomobile build -x github.com/udhos/fugo/demo/invader

9\. Push into Android device

Recipe:

    gomobile install github.com/udhos/fugo/demo/invader

## Building the ARENA server

10\. Build the server

Recipe:

    $ go install github.com/udhos/fugo/arena

10\. Run the server

    $ (cd demo/invader && arena)

The arena server needs to load image information from demo/invader/assets.

## How does the INVADER application locate the ARENA server?

The Invader application will continously try two methods to reach the server:

a) The Invader application will send a discovery request to UDP 239.1.1.1:8888. If there is an Arena server in the LAN, it will respond reporting its TCP endpoint. This local discovery is useful for quickly deploying a local Arena server. It depends on multicasting on the local network.

b) The Invader application will try to connect to the Arena server specified in the file server.txt:

    $ more demo/invader/assets/server.txt 
    localhost:8080

The TCP endpoint hard-coded in the file server.txt is included in the APK file. You will need to rebuild and redeploy the application to change it. This option is useful for deploying public Arena server on the Internet.

## INVADER runtime flags

You can tweak the app behavior by changing these files before gomobile build:

    demo/invader/assets/box.txt    - bool (file_exists=true)
    demo/invader/assets/server.txt - string host:port (TCP endpoint for server)
    demo/invader/assets/slow.txt   - bool (file_exists=true)
    demo/invader/assets/trace.txt  - string host:port (UDP endpoint for logs)

## KNOWN ISSUES

### x/mobile: build failing when using go modules

https://github.com/golang/go/issues/27234

### Need way to hide Android status bar. Fixed: add the theme below to AndroidManifest.xml

Add this to AndroidManifest.xml:

    <activity android:name="org.golang.app.GoNativeActivity"
        android:theme="@android:style/Theme.NoTitleBar.Fullscreen"
    </activity>

https://github.com/golang/go/issues/12766

https://github.com/golang/go/issues/21396

### Need way to set Android app icon. Fixed: add assets/icon.png

https://github.com/golang/go/issues/9985

https://golang.org/cl/30019

### Need way to call Android API from Go.

Reverse Binding https://www.slideshare.net/takuyaueda967/mobile-apps-by-pure-go-with-reverse-binding

slides 65-67 from https://pt.slideshare.net/takuyaueda967/go-for-mobile-games

#### RunOnJVM

RunOnJVM added to gomobile:

- https://github.com/golang/go/issues/26815 

- https://golang.org/cl/127758

Old info on RunOnJVM: https://gist.github.com/tenntenn/aae3d14d0df4884ac4e7

## References

### Just port a Golang game to Android

https://dev.to/ntoooop/just-port-a-golang-game-to-android--3a9f

### Korok Game Engine

https://korok.io/

https://github.com/KorokEngine/Korok

--xx--

