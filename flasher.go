package main

import (
	"archive/zip"
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
)

var executable, _ = os.Executable()
var cwd = filepath.Dir(executable)

const OS = runtime.GOOS
const PLATFORM_TOOLS_ZIP = "platform-tools-latest-" + OS + ".zip"

var adb = exec.Command("adb")
var fastboot = exec.Command("fastboot")

var factoryImage string
var altosImage string
var altosKey string
var devices []string

func main() {
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
				continue
			}
			if strings.Contains(file, "altos-") {
				altosImage = file
				continue
			}
		} else if strings.HasSuffix(file, ".bin") {
			altosKey = file
			continue
		}
	}
	if altosImage == "" {
		fmt.Println("Cannot continue without altOS device image")
		os.Exit(1)
	}
	err = checkPlatformTools()
	if err != nil {
		fmt.Println("There are missing Android platform tools in PATH. Attempting to download them from https://dl.google.com/android/repository/" + PLATFORM_TOOLS_ZIP)
		err := getPlatformTools()
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Cannot continue without Android platform tools. Exiting...")
			os.Exit(1)
		}
	}
	getDevices()
	if factoryImage == "" {
		fmt.Println("Factory image missing. Attempting to download from https://developers.google.com/android/images/index.html")
		err = getFactoryImage()
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("Cannot continue without the device factory image. Exiting...")
			os.Exit(1)
		}
	}
	flashDevices()
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
	platformToolCommand := *adb
	platformToolCommand.Args = append(adb.Args, "-s", devices[0], "shell", "getprop", "|", "grep", "ro.product.device", "|", "awk", "'{print $2}'")
	out, err := platformToolCommand.Output()
	device := string(out)
	device = strings.Trim(device, "[]\n")
	if device == "" {
		return err
	}
	resp, err := http.Get("https://developers.google.com/android/images/index.html")
	if err != nil {
		return err
	}
	out, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(out)
	//TODO remove device = sargo
	device = "sargo"
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
	err = extractZip(factoryImage, cwd)
	return nil
}

func flashDevices() {
	var wg sync.WaitGroup
	for _, device := range devices {
		go func(device string) {
			defer wg.Done()
			log.Println("Flashing device " + device)
			err := exec.Command("."+string(os.PathSeparator)+"flasher.sh", "-s "+device).Run()
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
