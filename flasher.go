package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var executable, _ = os.Executable()
var cwd = filepath.Dir(executable)

const OS = runtime.GOOS
const PLATFORM_TOOLS_ZIP = "platform-tools-latest-" + OS + ".zip"

var adb = exec.Command("adb")
var fastboot = exec.Command("fastboot")

var input string

var factoryImage string
var bootloader string
var radio string
var image string
var altosImage string
var altosKey string
var avbVersion string
var devices []string

func main() {
	checkPrerequisiteFiles()
	err := checkPlatformTools()
	if err != nil {
		fmt.Println("There are missing Android platform tools in PATH. Attempting to download https://dl.google.com/android/repository/" + PLATFORM_TOOLS_ZIP)
		err := getPlatformTools()
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Cannot continue without Android platform tools. Exiting...")
			os.Exit(1)
		}
	}

	fmt.Println("Do the following for each device:")
	fmt.Println("Enable Developer Options on device (Settings -> About Phone -> tap \"Build number\" 7 times)")
	fmt.Println("Enable USB debugging on device (Settings -> System -> Advanced -> Developer Options) and allow the computer to debug (hit \"OK\" on the popup when USB is connected)")
	fmt.Println("Enable OEM Unlocking (in the same Developer Options menu)")
	fmt.Print("When done, press enter to continue")
	_, _ = fmt.Scanln(&input)

	getDevices()
	if len(devices) == 0 {
		fmt.Println("No device connected. Exiting...")
		os.Exit(1)
	}
	avbVersion = getProp("ro.boot.avb_version")
	if factoryImage == "" {
		fmt.Println("Factory image missing. Attempting to download from https://developers.google.com/android/images/index.html")
		err = getFactoryImage()
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Cannot continue without the device factory image. Exiting...")
			os.Exit(1)
		}
	}
	err = extractZip(factoryImage, cwd)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Cannot continue without the device factory image. Exiting...")
		os.Exit(1)
	}
	factoryImage = regexp.MustCompile(".*\\.[0-9]{3}").FindAllString(factoryImage, -1)[0]
	factoryImage = cwd + string(os.PathSeparator) + factoryImage + string(os.PathSeparator)
	files, err := ioutil.ReadDir(factoryImage)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Cannot continue without the device factory image. Exiting...")
		os.Exit(1)
	}
	for _, file := range files {
		file := file.Name()
		if strings.Contains(file, "bootloader") {
			bootloader = factoryImage + file
		} else if strings.Contains(file, "radio") {
			radio = factoryImage + file
		} else if strings.Contains(file, "image") {
			image = factoryImage + file
		}
	}
	flashDevices()
}

func checkPrerequisiteFiles() {
	files, err := ioutil.ReadDir(cwd)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	for _, file := range files {
		file := file.Name()
		if strings.HasSuffix(file, ".zip") {
			if strings.Contains(file, "factory") {
				factoryImage = file
			} else if strings.Contains(file, "-img-") {
				altosImage = file
			}
		} else if strings.HasSuffix(file, ".bin") {
			altosKey = file
		}
	}
	if altosImage == "" {
		fmt.Println("Cannot continue without altOS device image. Exiting...")
		os.Exit(1)
	}
}

func getDevices() {
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "devices")
	output, _ := platformToolCommand.Output()
	devices = strings.Split(string(output), "\n")
	devices = devices[1 : len(devices)-2]
	for i, device := range devices {
		device = strings.Split(device, "\t")[0]
		devices[i] = device
	}
}

func getFactoryImage() error {
	device := getProp("ro.product.device")
	if device == "" {
		return errors.New("could not find prop ro.product.device")
	}
	resp, err := http.Get("https://developers.google.com/android/images/index.html")
	if err != nil {
		return err
	}
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(out)
	links := regexp.MustCompile("http.*("+device+"-p).*([0-9]{3}-).*(.zip)").FindAllString(body, -1)
	link := links[len(links)-1]
	_, err = url.ParseRequestURI(link)
	if err != nil {
		return err
	}
	factoryImage = path.Base(link)
	err = downloadFile(link, factoryImage)
	if err != nil {
		return err
	}
	return nil
}

func getProp(prop string) string {
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "-s", devices[0], "shell", "getprop", "|", "grep", prop, "|", "awk", "'{print $2}'")
	out, err := platformToolCommand.Output()
	if err != nil {
		return ""
	}
	prop = string(out)
	prop = strings.Trim(prop, "[]\n\r")
	return prop
}

func flashDevices() {
	for _, device := range devices {
		platformToolCommand := *adb
		platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "reboot", "bootloader")
		err := platformToolCommand.Run()
		if err != nil {
			log.Println(err.Error())
			return
		}
		time.Sleep(5 * time.Second)
		fmt.Println("Unlocking device " + device + " bootloader...")
		fmt.Println("Please use the volume and power keys on the device to confirm.")
		platformToolCommand = *fastboot
		platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "flashing", "unlock")
		err = platformToolCommand.Run()
		if err != nil {
			log.Println(err.Error())
			return
		}
		time.Sleep(5 * time.Second)
		fmt.Print("Press enter to continue")
		_, _ = fmt.Scanln(&input)
	}
	var wg sync.WaitGroup
	for _, device := range devices {
		wg.Add(1)
		go func(device string) {
			defer wg.Done()
			fmt.Println("Flashing stock firmware on device " + device + "...")
			platformToolCommand := *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "--slot", "all", "flash", "bootloader", bootloader)
			err := platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "reboot-bootloader")
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "--slot", "all", "flash", "radio", radio)
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "reboot-bootloader")
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "--skip-reboot", "update", image)
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "reboot-bootloader")
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			fmt.Println("Flashing altOS on device " + device)
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "--skip-reboot", "update", altosImage)
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			if altosKey != "" {

			}
			fmt.Println("Wiping userdata for device " + device + "...")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "-w", "reboot-bootloader")
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			if altosKey == "" || avbVersion == "" {
				fmt.Println("Done flashing device " + device + "... Rebooting")
				platformToolCommand = *fastboot
				platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "reboot")
				err = platformToolCommand.Run()
				if err != nil {
					log.Println(err.Error())
					return
				}
			}
		}(device)
	}
	wg.Wait()
	if altosKey != "" && avbVersion != "" {
		for _, device := range devices {
			platformToolCommand := *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "flash avb_custom_key", altosKey)
			err := platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			fmt.Println("Unlocking device " + device + " bootloader...")
			fmt.Println("Please use the volume and power keys on the device to confirm.")
			platformToolCommand = *fastboot
			platformToolCommand.Args = append(platformToolCommand.Args, "-s", device, "flashing", "lock")
			err = platformToolCommand.Run()
			if err != nil {
				log.Println(err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			fmt.Print("Press enter to continue")
			_, _ = fmt.Scanln(&input)
		}
	}
	fmt.Println("Bulk flashing complete")
}

func getPlatformTools() error {
	err := downloadFile("https://dl.google.com/android/repository/"+PLATFORM_TOOLS_ZIP, PLATFORM_TOOLS_ZIP)
	if err != nil {
		return err
	}
	err = extractZip(PLATFORM_TOOLS_ZIP, cwd)
	if err != nil {
		return err
	}
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

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %s downloaded", Bytes(wc.Total))
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func Bytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
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

	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	fmt.Println()
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
