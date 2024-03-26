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
	"time"

	"github.com/panjf2000/ants/v2"
)

var list sync.Map
var pool *ants.Pool
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
	pool, _ = ants.NewPool(goruntineNum, ants.WithPreAlloc(true))
	defer pool.Release()
	if modPath == "" && packagePath == "" {
		fmt.Println("Please input mod file path or package path")
		return
	}
	wg = sync.WaitGroup{}
	if packagePath == "" {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			downloadModFileAndParseJson(modPath)
		})
	}
	if modPath == "" {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			downloadPackageAndParseJson(packagePath)
		})
	}
	wg.Wait()
	fmt.Println("------------------------")
	fmt.Printf("\t\tDone\n")
	fmt.Println("------------------------")
}

func downloadModFileAndParseJson(modPath string) {
	// ch <- struct{}{}
	// defer func() { <-ch }()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	defer func() {
		//get modPath dir and removeall dir
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
	fmt.Println("start exec shell:", shell)
	now := time.Now()
	cmd := exec.Command("sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(output))
		return
	}
	fmt.Println("end exec shell:", shell)
	fmt.Println("exec time:", time.Since(now))
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
				fmt.Println(out)
				return
			}
			out = ""
			if _, ok := list.Load(mod.GoMod); ok {
				continue
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
			//copy mod file to path
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				downloadModFileAndParseJson(targetFile)
			})
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
				fmt.Println(err)
				return
			}
			out = ""
			if _, ok := list.Load(mod.GoMod); ok {
				continue
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
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				downloadModFileAndParseJson(targetFile)
			})
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
