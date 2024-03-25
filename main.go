package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"sync"
)

type Mod struct {
	Path     string
	Version  string
	Info     string
	GoMod    string
	Zip      string
	Dir      string
	Sum      string
	GoModSum string
}

func main() {
	var modPath string
	var goruntineNum int
	var packagePath string
	flag.StringVar(&modPath, "m", "", "modPath")
	flag.StringVar(&packagePath, "p", "", "packagePath")
	flag.IntVar(&goruntineNum, "n", 10, "goruntineNum")
	flag.Parse()
	if modPath == "" && packagePath == "" {
		fmt.Println("Please input mod file path or package path")
		return
	}
	ch := make(chan struct{}, goruntineNum)
	wg := sync.WaitGroup{}
	if packagePath == "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloadModFileAndParseJson(modPath, ch)
		}()
	}
	if modPath == "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloadPackageAndParseJson(packagePath, ch)
		}()
	}
	wg.Wait()
}

func downloadModFileAndParseJson(modPath string, ch chan struct{}) {
	ch <- struct{}{}
	defer func() { <-ch }()
	shell := fmt.Sprintf("go mod download -json -modfile=%s", modPath)
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(output))
		return
	}
	fmt.Println(string(output))
	var out string
	buf := bytes.Buffer{}
	buf.Write(output)
	reader := bufio.NewReader(&buf)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		out += string(line)
		if string(line) == "}" {
			var mod Mod
			err := json.Unmarshal([]byte(out), &mod)
			if err != nil {
				fmt.Println(err)
				return
			}
			out = ""
			fmt.Println(mod.GoMod)
			downloadModFileAndParseJson(mod.GoMod, ch)
		}
	}
}

func downloadPackageAndParseJson(packagePath string, ch chan struct{}) {
	ch <- struct{}{}
	defer func() { <-ch }()
	shell := fmt.Sprintf("go list -m -json %s", packagePath)
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(output))
	var out string
	buf := bytes.Buffer{}
	buf.Write(output)
	reader := bufio.NewReader(&buf)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		out += string(line)
		if string(line) == "}" {
			var mod Mod
			err := json.Unmarshal([]byte(out), &mod)
			if err != nil {
				fmt.Println(err)
				return
			}
			out = ""
			fmt.Println(mod.GoMod)
			downloadModFileAndParseJson(mod.GoMod, ch)
		}
	}
}
