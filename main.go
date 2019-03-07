package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type File struct {
	Name string `json:"name" binding:"required"`
	Size int64  `json:"size" binding:"required"`
	Url  string `json:"url" binding:"required"`
}

type Release struct {
	Name        string    `json:"name" binding:"required"`
	Version     string    `json:"version" binding:"required"`
	Tag         string    `json:"tag" binding:"required"`
	Draft       bool      `json:"draft"`
	News        string    `json:"news"`
	PreRelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Files       []File    `json:"files"`
}

func ReleaseVersion(release Release) {
	var pathPrefix string
	var overwrite bool

	version_list := strings.Split(release.Version, ".")
	api_version := strings.Join(version_list[:2], ".")
	if release.Draft {
		pathPrefix = path.Join(config.Path.Release, api_version)
		overwrite = true
	} else if release.PreRelease {
		pathPrefix = path.Join(config.Path.Release, api_version)
		overwrite = true
	} else {
		pathPrefix = path.Join(config.Path.Release, api_version)
		overwrite = false
	}

	for _, file := range release.Files {
		filepath := path.Join(pathPrefix, file.Name)
		if err := DownloadFile(file.Url, filepath, overwrite, file.Size); err != nil {
			fmt.Printf("Download \"%s\" Error: %s\n", file.Url, err)
		}
	}

	newsfile := fmt.Sprintf("%s-%s.news", release.Name, release.Version)
	newspath := path.Join(pathPrefix, newsfile)
	if strings.Contains(release.News, "Changes since the last release:") {
		lines := strings.Split(release.News, "\n")
		news := strings.Join(lines[1:], "\n") + "\n"
		ioutil.WriteFile(newspath, []byte(news), 0644)
	} else {
		ioutil.WriteFile(newspath, []byte(release.News+"\n"), 0644)
	}
	return
}

func NonceIsNew(nonce string) bool {
	//TODO
	return true
}

func checkValid(header http.Header, body []byte) bool {
	nonce := header.Get("X-Build-Nonce")
	if !NonceIsNew(nonce) {
		return false
	}

	if config.Secret.ApiSecret == "" {
		return true
	}

	data := nonce + string(body)
	result := calcHmac(config.Secret.ApiSecret, data)
	signature := header.Get("X-Build-Signature")
	s1, s2 := strings.ToLower(signature), strings.ToLower(result)

	if s1 == s2 {
		return true
	} else {
		return false
	}
}

func Update(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("Error reading body: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "can't read body."})
		return
	}

	if len(config.Secret.AllowIps) > 0 {
		if !isIpAllowed(c.Request, config.Secret.AllowIps) {
			c.JSON(http.StatusForbidden, gin.H{"status": "invalid ip addr."})
			return
		}
	}

	if !checkValid(c.Request.Header, body) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "invalid request."})
		return
	}

	var release Release
	err = json.Unmarshal(body, &release)
	if err != nil {
		fmt.Errorf("Can not decode data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "can't decode data"})
		return
	}

	if len(config.Secret.AllowRepos) > 0 {
		if !isRepoAllowed(release) {
			c.JSON(http.StatusForbidden, gin.H{"status": "disallow release the software"})
			return
		}
	}

	ReleaseVersion(release)
	c.JSON(http.StatusOK, gin.H{"status": "release success"})
}

func main() {
	router := gin.Default()
	router.POST("/post", Update)
	gin.SetMode(gin.ReleaseMode)

	http.ListenAndServe(":"+strconv.Itoa(config.Web.Port), router)
}
