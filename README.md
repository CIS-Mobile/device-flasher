# Dependencies
macOS:
```
brew install go
```
Ubuntu:
```
sudo apt install golang
```
Fedora:
```
sudo dnf install go
```
Windows
```
winget install --id=GoLang.Go  -e
```

# Compile
```
go mod init cissecure.com/device-flasher
go mod tidy
make
```

# Usage
Plug each device of the same model to a USB port on the machine the program is running from.
> You must copy the altOS factory image for the device you want to install altOS on to the current directory.

## macOS:
Open a terminal in the current directory and enter the following command:
```
./altOS-flasher_darwin-universal
```
## Linux:
Open a terminal in the current directory and enter the following command:
```
sudo ./altOS-flasher_linux-x86_64
```
## Windows:
Open [Windows Terminal](https://learn.microsoft.com/en-us/windows/terminal/install) in the current directory and select the Powershell profile. Enter the following command:
```
./altOS-flasher_windows-x86_64.exe
```

Alternatively you can simply double click on the executable, but note this will not show any error output if there is any.
