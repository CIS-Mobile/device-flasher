$VERSION = git describe --abbrev=0 --tags

$GIT_COMMIT = git rev-list -1 HEAD

function Build-GoProject {
    param (
        [string]$GOOS,
        [string]$GOARCH,
        [string]$outputName
    )

    Write-Host "Building for OS: $GOOS, Arch: $GOARCH, Output: $outputName"

    $env:GO111MODULE = "auto"
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "0"

    go build -o $outputName -ldflags "-X main.version=$VERSION -X 'main.gitCommit=$GIT_COMMIT'" .
}

Build-GoProject -GOOS "linux" -GOARCH "amd64" -outputName "altOS-flasher_linux-x86_64"
Build-GoProject -GOOS "windows" -GOARCH "amd64" -outputName "altOS-flasher_windows-x86_64.exe"
Build-GoProject -GOOS "darwin" -GOARCH "amd64" -outputName "altOS-flasher_darwin-x86_64"
Build-GoProject -GOOS "darwin" -GOARCH "arm64" -outputName "altOS-flasher_darwin-arm64"