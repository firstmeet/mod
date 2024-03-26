package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var list sync.Map
var wg sync.WaitGroup

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
	if packagePath == "" {
		downloadModFileAndParseJson(modPath)
	}
	if modPath == "" {
		downloadPackageAndParseJson(packagePath)
	}
	wg.Wait()
	fmt.Println("------------------------")
	fmt.Printf("\t\t\t\tDone\n")
	fmt.Println("------------------------")
}
func downloadModFileAndParseJson(modPath string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	defer func() {
		dir := filepath.Dir(modPath)
		if dir == "." {
			return
		}
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Println(err)
		}
	}()
	shell := fmt.Sprintf("go mod download -json -modfile=%s", modPath)
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(output))
		return
	}
	var out string
	buf := bytes.Buffer{}
	buf.Write(output)
	reader := bufio.NewReader(&buf)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println(err)
			return
		}
		out += string(line)
		if string(line) == "}" {
			var mod Mod
			err := json.Unmarshal([]byte(out), &mod)
			if err != nil {
				fmt.Println(err)
				fmt.Println(out)
				return
			}
			out = ""
			if _, ok := list.Load(mod.GoMod); ok {
				return
			}
			list.Store(mod.GoMod, struct{}{})
			err = os.MkdirAll(mod.Path+"/"+mod.Version, 0755)
			if err != nil {
				fmt.Println(err)
				return
			}
			targetFile := filepath.Join(mod.Path, mod.Version, "go.mod")
			_, err = CopyFile(mod.GoMod, targetFile)
			if err != nil {
				fmt.Println(err)
				return
			}
			downloadModFileAndParseJson(targetFile)
		}
	}

}
func downloadPackageAndParseJson(packagePath string) {
	// ch <- struct{}{}
	// defer func() { <-ch }()
	shell := fmt.Sprintf("go list -m -json %s", packagePath)
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return
	}
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
				out = ""
				fmt.Println(err)
				return
			}
			out = ""
			if _, ok := list.Load(mod.GoMod); ok {
				return
			}
			list.Store(mod.GoMod, struct{}{})
			err = os.MkdirAll(mod.Path+"/"+mod.Version, 0755)
			if err != nil {
				fmt.Println(err)
				return
			}
			targetFile := filepath.Join(mod.Path, mod.Version, "go.mod")
			_, err = CopyFile(mod.GoMod, targetFile)
			if err != nil {
				fmt.Println(err)
				return
			}
			downloadModFileAndParseJson(targetFile)
		}
	}
}
func CopyFile(sourceFile, targetFile string) (written int64, err error) {
	src, err := os.Open(sourceFile)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.Create(targetFile)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
