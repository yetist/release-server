package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
)

func calcHmac(secret, data string) (result string) {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	result = hex.EncodeToString(h.Sum(nil))
	return
}

func AllowRepo(url string) bool {
	allow := false
	for _, v := range config.Security.AllowRepos {
		prefix := "https://github.com/" + v + "/releases/download"
		if v == "*" || strings.HasPrefix(url, prefix) {
			allow = true
		}
	}
	return allow
}

func isIpAllowed(r *http.Request, AllowIp []string) (authorized bool) {
	var ip string
	if ipProxy := r.Header.Get("X-Real-IP"); len(ipProxy) > 0 {
		ip = ipProxy
	} else if ipProxy := r.Header.Get("X-FORWARDED-FOR"); len(ipProxy) > 0 {
		ip = ipProxy
	} else {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	rAddr := net.ParseIP(ip)
	for _, v := range AllowIp {
		_, ipNet, err := net.ParseCIDR(v)
		if err != nil {
			ipHost := net.ParseIP(v)
			if ipHost != nil {
				if ipHost.Equal(rAddr) {
					authorized = true
				}
			}
		} else {
			if ipNet.Contains(rAddr) {
				authorized = true
			}
		}
	}
	return
}

func isRepoAllowed(release Release) bool {
	allow := false
	for _, file := range release.Files {
		allow = AllowRepo(file.Url)
	}
	return allow
}

func download(filepath, url string) (err error) {
	fmt.Println("Downloading ", url, " to ", filepath)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	f, err := os.Create(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return
}

func DownloadFile(url, filepath string, overwrite bool, size int64) (err error) {
	dirname := path.Dir(filepath)
	os.MkdirAll(dirname, 0755)

	if IsFile(filepath) {
		if !overwrite {
			return errors.New("File already exists.")
		}
	}

	if err = download(filepath, url); err != nil {
		return
	}
	if size != get_file_size(filepath) {
		return errors.New("File size is different.")
	}
	return nil
}

func WriteFile(path string, data []byte) error {
	fmt.Printf("WriteFile: Size of download: %d\n", len(data))
	return ioutil.WriteFile(path, data, 0644)
}

func LinkFile(srcName, dstName string) error {
	dirname := path.Dir(dstName)
	os.MkdirAll(dirname, 0755)

	return os.Symlink(srcName, dstName)
}

func CopyFile(srcName, dstName string) (written int64, err error) {
	dirname := path.Dir(dstName)
	os.MkdirAll(dirname, 0755)

	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)
}

func get_file_size(path string) (size int64) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return
	}
	size = stat.Size()
	return
}
