@echo off
Rem This Windows Batch file will build the Windows version of the Flasher tool.

FOR /F "tokens=*" %%g IN ('git rev-list -1 HEAD') do (SET GIT_COMMIT=%%g)

go build -ldflags "-X main.version=('git describe --abbrev=0 --tags') -X 'main.gitCommit=%GIT_COMMIT%'" .