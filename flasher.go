package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const OS = runtime.GOOS
const PLATFORM_TOOLS_ZIP = "platform-tools-latest-" + OS + ".zip"

var adb = exec.Command("adb")
var fastboot = exec.Command("fastboot")

func main() {
	err := checkPlatformTools()
	if err != nil {
		fmt.Println("There are missing Android platform tools in PATH. Attempting to download them from https://dl.google.com/android/repository/" + PLATFORM_TOOLS_ZIP)
		err := getPlatformTools()
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Cannot continue without Android platform tools. Exiting...")
			os.Exit(1)
		}
	}
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "devices")
	output, _ := platformToolCommand.Output()
	devices := strings.Split(string(output), "\n")
	devices = devices[1:len(devices)-2]
	var wg sync.WaitGroup
	for i, device := range devices {
		wg.Add(1)
		device = strings.Split(device, "\t")[0]
		devices[i] = device
		go func(device string) {
			defer wg.Done()
			log.Println("Flashing device " + device)
			err = exec.Command("." + string(os.PathSeparator) + "flasher.sh", "-s " + device).Run()
			if err != nil {
				log.Println("Flashing failed for device " + device + " with error: " + err.Error())
			}
		}(device)
	}
	wg.Wait()
	fmt.Println("Done")
}

func getPlatformTools() error {
	err := downloadFile("https://dl.google.com/android/repository/platform-tools-latest-"+OS+".zip", PLATFORM_TOOLS_ZIP)
	if err != nil {
		return err
	}
	cwd, err := os.Executable()
	cwd = filepath.Dir(cwd)
	err = extractZip(PLATFORM_TOOLS_ZIP, cwd)
	platformToolsPath := cwd + string(os.PathSeparator) + "platform-tools" + string(os.PathSeparator)
	adbPath := platformToolsPath + "adb"
	fastbootPath := platformToolsPath + "fastboot"
	if OS == "windows" {
		adbPath += ".exe"
		fastbootPath += ".exe"
	}
	adb = exec.Command(adbPath)
	fastboot = exec.Command(fastbootPath)
	return err
}

func extractZip(src, dest string) error {
	dest = filepath.Clean(dest) + string(os.PathSeparator)

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		path := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(path, dest) {
			return fmt.Errorf("%s: illegal file path", path)
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
		} else {
			os.MkdirAll(filepath.Dir(path), 0755)
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func checkPlatformTools() error {
	err := checkAdb()
	if err != nil {
		return err
	}
	return checkFastboot()
}

func checkAdb() error {
	platformTool := *adb
	platformTool.Args = append(platformTool.Args, "version")
	return checkCommand(platformTool)
}

func checkFastboot() error {
	platformTool := *fastboot
	platformTool.Args = append(platformTool.Args, "--version")
	return checkCommand(platformTool)
}

func checkCommand(platformTool exec.Cmd) error {
	_, err := platformTool.Output()
	if err != nil {
		return err
	}
	return nil
}
