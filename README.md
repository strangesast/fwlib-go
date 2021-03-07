# fwlib & go

Lazy and deliberately minimal example of fwlib called from go. Surprisingly painless.

## Instructions

0. clone submodules (fwlib) `git submodule update --init --recursive`
1. install appropriate fwlib (arm/x86/x64) to `/usr/local/lib` or symlink with `ln -s libfwlib32-linux-$ARCH.so.1.0.5 libfwlib32.so`
2. get dependencies `go get`
3. build `go build`
