package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	battleNetInstaller = "https://downloader.battle.net/download/getInstallerForGame?os=win&gameProgram=BATTLENET_APP&version=Live"
	hdtInstaller = "https://github.com/HearthSim/Hearthstone-Deck-Tracker/releases/download/v1.23.15/Hearthstone.Deck.Tracker-v1.23.15.zip"
)

func main(){
	//check app data
	init, err := checkInit()
	if err != nil {
		fmt.Printf("Error: %s", err)
		time.Sleep(15 * time.Second)
		os.Exit(1)
	}

	//If not installed - installed
	if !init {
		fmt.Printf("Games not isntalled. Installing...\n")
		err := installBins()
		if err != nil {
			fmt.Printf("Error: %s", err)
			time.Sleep(15 * time.Second)
			os.Exit(1)
		}
	} else {
		err := launchBins()
		if err != nil {
			fmt.Printf("Error: %s", err)
			time.Sleep(15 * time.Second)
			os.Exit(1)
		}
	}

	//if installed. Launch battlenet and hdt

}

//Check if installed file is in the user config directory
func checkInit() (bool, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf("%s%c%s%cinstalled", configDir, os.PathSeparator, "HearthstoneNerdLinux", os.PathSeparator)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist){
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

//download and loaunch battlenet installer and install HDT
func installBins() error {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	errs := make([]error, 2)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return  err
	}

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		fmt.Printf("Installing batlle.net\n")
		file := fmt.Sprintf("%s%cbattlenet.exe", cacheDir, os.PathSeparator)
		errs[0] = downloadBin(battleNetInstaller, file)
		if errs[0] != nil {
			return
		}

		cmd := exec.Command(file)
		errs[0] = cmd.Run()
	}(wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		fmt.Printf("Installing hdt\n")
		file := fmt.Sprintf("%s%chdt.zip", cacheDir, os.PathSeparator)
		errs[1] = downloadBin(hdtInstaller, file)
		if errs[1] != nil {
			return
		}

		programDir := "C:\\Program Files (x86)\\hnl"
		errs[1] = unzip(file, programDir)
		if errs[1] != nil {
			return
		}
	}(wg)

	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(configDir, "HearthstoneLinuxNerd"), 0755)
	if err != nil {
		return err
	}

	_, err = os.Create(filepath.Join(configDir, "HearthstoneLinuxNerd", "installed"))
	if err != nil {
		return err
	}

	return nil
}

func launchBins() error {
	battleNetBin := filepath.Join("C:", "Program Files (x86)", "Battle.net", "Battle.net.exe")
	hdtBin := filepath.Join("C:", "Program Files (x86)", "hnl", "Hearthstone Deck Tracker", "Hearthstone Deck Tracker.exe")

	wg := new(sync.WaitGroup)
	wg.Add(2)
	errs := make([]error, 2)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		cmd := exec.Command(battleNetBin)
		errs[0] = cmd.Run()
	}(wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		cmd := exec.Command(hdtBin)
		errs[1] = cmd.Run()
	}(wg)

	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

func downloadBin(uri, filepath string) error {// Build fileName from fullPath
    fileURL, err := url.Parse(uri)
    if err != nil {
		return err
    }

    // Create blank file
	file, err := os.Create(filepath)
    if err != nil {
		return err
    }
    client := http.Client{
        CheckRedirect: func(r *http.Request, via []*http.Request) error {
            r.URL.Opaque = r.URL.Path
            return nil
        },
    }
    // Put content on file
    resp, err := client.Get(fileURL.String())
    if err != nil {
		return err
    }
    defer resp.Body.Close()

    _, err = io.Copy(file, resp.Body)

    defer file.Close()

	return err
}

func unzip(src, dest string) error {
    r, err := zip.OpenReader(src)
    if err != nil {
        return err
    }
    defer func() {
        if err := r.Close(); err != nil {
			fmt.Printf("stuff...\n")
			time.Sleep(10)
            panic(err)
        }
    }()

    os.MkdirAll(dest, 0755)

    // Closure to address file descriptors issue with all the deferred .Close() methods
    extractAndWriteFile := func(f *zip.File) error {
        rc, err := f.Open()
        if err != nil {
            return err
        }
        defer func() {
            if err := rc.Close(); err != nil {
				fmt.Printf("stuff...\n")
				time.Sleep(10)
				panic(err)
            }
        }()

        path := filepath.Join(dest, f.Name)

        // Check for ZipSlip (Directory traversal)
        if !strings.HasPrefix(path, filepath.Clean(dest) + string(os.PathSeparator)) {
            return fmt.Errorf("illegal file path: %s", path)
        }

        if f.FileInfo().IsDir() {
            os.MkdirAll(path, f.Mode())
        } else {
            os.MkdirAll(filepath.Dir(path), f.Mode())
            f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
            if err != nil {
                return err
            }
            defer func() {
                if err := f.Close(); err != nil {
					fmt.Printf("stuff...\n")
					time.Sleep(10)
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
