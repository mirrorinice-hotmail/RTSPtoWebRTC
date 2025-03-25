@REM https://habr.com/ru/post/249449/

SET GOOS=windows
SET GOARCH=amd64
go build -ldflags "-s -w" -o ./rinortsp2web.exe

@SET GOOS=linux
@SET GOARCH=386
@REM go build -ldflags "-s -w" -o ./rtsp2webrtc_i386

SET GOOS=linux
SET GOARCH=amd64
go build -ldflags "-s -w" -o ./rinortsp2web

@SET GOOS=linux
@SET GOARCH=arm
@SET GOARM=7
@REM go build -ldflags "-s -w" -o ./rtsp2webrtc_armv7

@SET GOOS=linux
@SET GOARCH=arm64
@REM go build -ldflags "-s -w" -o ./rtsp2webrtc_aarch64

@SET GOOS=darwin
@SET GOARCH=amd64
@REM go build -ldflags "-s -w" -o ./rtsp2webrtc_darwin
