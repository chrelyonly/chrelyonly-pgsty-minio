set ANDROID_NDK=D:\dev\sdk\andorid\ndk\28.2.13676358
set TOOLCHAIN=%ANDROID_NDK%\toolchains\llvm\prebuilt\windows-x86_64
set GOOS=android
set GOARCH=amd64
set API=26
set CGO_ENABLED=1
set CC=%TOOLCHAIN%\bin\aarch64-linux-android%API%-clang 
set CXX=%TOOLCHAIN%\bin\aarch64-linux-android%API%-clang++