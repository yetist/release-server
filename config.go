package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

type Config struct {
	Web struct {
		Debug bool   `toml:"debug"`
		Host  string `toml:"host"`
		Port  int
		Log   string `toml:"log"`
	}
	Path struct {
		Release       string `toml:"release"`
		PreRelease    string `toml:"pre-release"`
		Draft         string `toml:"draft"`
		Source        string `toml:"source"`
		SourceSymlink bool   `toml:"symlink_in_source"`
	}
	Rss struct {
		Title       string `toml:"title"`
		Description string `toml:"description"`
		Link        string `toml:"link"`
		Path        string `toml:"path"`
		Count       int    `toml:"count"`
		UrlPrefix   string `toml:"url_prefix"`
	}
	Security struct {
		ApiSecret  string   `toml:"api_secret"`
		AllowRepos []string `toml:"allow_repos"`
		AllowIps   []string `toml:"allow_ips"`
	}
	Mail struct {
		Host       string `toml:"smtp_host"`
		Port       int    `toml:"smtp_port"`
		Username   string
		Password   string
		Sender     string
		SenderNick string `toml:"sender_nick"`
		Receivers  []string
	}
}

var defConfig = `
#
[web]
#http server host and port
debug = false
host = "localhost"
port = 9090
log = "/tmp/release-server.log"

[path]
# path to save release version tarballs.
release = "/tmp/release"

# path to save pre-release version tarballs.
pre-release = "/tmp/prerelease"

# path to save draft version tarballs.
draft = "/tmp/draft"

# path to save the release version tarballs as source.
source = "/tmp/sources"

# create symlink to other directory under source directory, default is false.
# symlink_in_source = true

[rss]
# Rss Channel title
title = "MATE releases"

# Rss Channel Description
description = "RSS feed for MATE releases"

# Rss Channel Link URL
link = "https://pub.mate-desktop.org/rss.xml"

# Rss file path, if path is not "", update the rss.xml file
path = "/tmp/rss.xml"

# Rss item counts
count = 30

# Rss item download parent url
url_prefix = "https://pub.mate-desktop.org/sources/{{.Name}}/{{.ApiVersion}}"

[security]
# secret key
# You need to set a hidden variable named API_SECRET on the travis ci and set the same value.
api_secret = "it is a secret string"

# allow repos
# only github is supported, the format is: "organization/repo", "*" allows any repository.
allow_repos = [
"mate-desktop/marco",
"*",
]

# allow ips
# ip address of travis ci: https://docs.travis-ci.com/user/ip-addresses/
allow_ips = [
"127.0.0.1",
"::1",
]

# mail server
[mail]
smtp_host = ""
smtp_port = 587
username = ""
password = ""
sender_nick = "Notify"
sender = "nobody@example.com"
receivers = ["abc@example.com"]
`

var config Config

func init() {
	err := LoadConfig("release-server", "0.1.6", "release-server.toml")
	if err != nil {
		log.Fatalf("Config error: %s\n", err)
	}
	if config.Security.ApiSecret == "" {
		config.Security.ApiSecret = os.Getenv("API_SECRET")
	}
}

func ExecPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	p, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	return p, nil
}

// WorkDir returns absolute path of work directory.
func ExecDir() (string, error) {
	execPath, err := ExecPath()
	return path.Dir(strings.Replace(execPath, "\\", "/", -1)), err
}

// IsFile returns true if given path is a file,
// or returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// ExpandUser is a helper function that expands the first '~' it finds in the
// passed path with the home directory of the current user.
//
// Note: This only works on environments similar to bash.
func ExpandUser(path string) string {
	if u, err := user.Current(); err == nil {
		return strings.Replace(path, "~", u.HomeDir, -1)
	}
	return path
}

func selfConfigDir() string {
	if dir, err := ExecDir(); err != nil || strings.HasSuffix(dir, "_obj/exe") {
		wd, _ := os.Getwd()
		return wd
	} else {
		return dir
	}
}

func userConfigDir(name, version string) (pth string) {
	if pth = os.Getenv("XDG_CONFIG_HOME"); pth == "" {
		pth = ExpandUser("~/.config")
	}

	if name != "" {
		pth = filepath.Join(pth, name)
	}

	if version != "" {
		pth = filepath.Join(pth, version)
	}

	return pth
}

func sysConfigDir(name, version string) (pth string) {
	if pth = os.Getenv("XDG_CONFIG_DIRS"); pth == "" {
		pth = "/etc/xdg"
	} else {
		pth = ExpandUser(filepath.SplitList(pth)[0])
	}
	if name != "" {
		pth = filepath.Join(pth, name)
	}

	if version != "" {
		pth = filepath.Join(pth, version)
	}
	return pth
}

func LoadConfig(name, version, cfgname string) (err error) {
	sysconf := path.Join(sysConfigDir(name, version), cfgname)
	userconf := path.Join(userConfigDir(name, version), cfgname)
	selfconf := path.Join(selfConfigDir(), cfgname)
	if IsFile(selfconf) {
		if _, err = toml.DecodeFile(selfconf, &config); err != nil {
			return
		}
	} else if IsFile(userconf) {
		if _, err = toml.DecodeFile(userconf, &config); err != nil {
			return
		}
	} else if IsFile(sysconf) {
		if _, err = toml.DecodeFile(sysconf, &config); err != nil {
			return
		}
	} else {
		if _, err = toml.Decode(defConfig, &config); err != nil {
			return
		}
		fmt.Printf("\n*** Valid config files ***\n1. %s\n2. %s\n3. %s\n", selfconf, userconf, sysconf)
		fmt.Printf("\n*** Example for config ***\n%s\n", defConfig)
	}
	return nil
}
