package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	ApiVersion  string    `json:"-"`
	Files       []File    `json:"files"`
}

type Security struct {
	gorm.Model
	Nonce string `gorm:"type:varchar(100);unique"`
}

func DownloadTarballs(release Release) (err error) {
	var pathPrefix string
	var overwrite, updateSource bool
	var sourceDir string

	if release.Draft {
		if release.Name == "mate-themes" {
			pathPrefix = path.Join(config.Path.Draft, "themes", release.ApiVersion)
		} else {
			pathPrefix = path.Join(config.Path.Draft, release.ApiVersion)
		}
		overwrite = true
	} else if release.PreRelease {
		if release.Name == "mate-themes" {
			pathPrefix = path.Join(config.Path.PreRelease, "themes", release.ApiVersion)
		} else {
			pathPrefix = path.Join(config.Path.PreRelease, release.ApiVersion)
		}
		overwrite = true
	} else {
		if release.Name == "mate-themes" {
			pathPrefix = path.Join(config.Path.Release, "themes", release.ApiVersion)
		} else {
			pathPrefix = path.Join(config.Path.Release, release.ApiVersion)
		}
		overwrite = false
		if config.Path.Source != "" {
			updateSource = true
			sourceDir = path.Join(config.Path.Source, release.Name, release.ApiVersion)
		}
	}

	for _, file := range release.Files {
		filepath := path.Join(pathPrefix, file.Name)
		if err = DownloadFile(file.Url, filepath, overwrite, file.Size); err != nil {
			log.Printf("Download \"%s\" Error: %s\n", file.Url, err)
		}
		if updateSource {
			dstpath := path.Join(sourceDir, file.Name)
			if config.Path.SourceSymlink {
				LinkFile(filepath, dstpath)
			} else {
				CopyFile(filepath, dstpath)
			}
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
	if updateSource {
		dstpath := path.Join(sourceDir, newsfile)
		if config.Path.SourceSymlink {
			LinkFile(newspath, dstpath)
		} else {
			CopyFile(newspath, dstpath)
		}

		if config.Rss.Path != "" {
			updateFeed(release)
		}
	}
	return
}

func NonceIsNew(nonce string) bool {
	db, err := gorm.Open("sqlite3", ExpandUser("~/.release.db"))
	if err != nil {
		return true
	}
	defer db.Close()

	db.AutoMigrate(&Security{})

	security := Security{Nonce: nonce}

	if db.NewRecord(security) {
		db.Create(&security)
		return true
	} else {
		return false
	}
}

func checkValid(header http.Header, body []byte) bool {
	nonce := header.Get("X-Build-Nonce")
	if !NonceIsNew(nonce) {
		log.Printf("ERROR: Nonce %s is old.\n", nonce)
		return false
	}

	if config.Security.ApiSecret == "" {
		log.Printf("Server ignore Client's API_SECRET\n")
		return true
	}

	data := nonce + string(body)
	result := calcHmac(config.Security.ApiSecret, data)
	signature := header.Get("X-Build-Signature")
	s1, s2 := strings.ToLower(signature), strings.ToLower(result)

	if s1 == s2 {
		return true
	} else {
		log.Printf("ERROR: Signature: Need %s, but get %s.\n", s2, s1)
		return false
	}
}

func PostRelease(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("ERROR: reading body: %v\n", err)
		c.JSON(http.StatusOK, gin.H{"status": "can't read body."})
		return
	}

	if len(config.Security.AllowIps) > 0 {
		if !isIpAllowed(c.Request, config.Security.AllowIps) {
			log.Printf("ERROR: Server dis-allow this IP Addr.\n")
			c.JSON(http.StatusForbidden, gin.H{"status": "invalid ip addr."})
			return
		}
	}

	if !checkValid(c.Request.Header, body) {
		log.Printf("ERROR: Invalid request.\n")
		c.JSON(http.StatusBadRequest, gin.H{"status": "invalid request."})
		return
	}

	var release Release
	err = json.Unmarshal(body, &release)
	if err != nil {
		log.Printf("Can not decode data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "can't decode data"})
		return
	}

	version_list := strings.Split(release.Version, ".")
	release.ApiVersion = strings.Join(version_list[:2], ".")

	if len(config.Security.AllowRepos) > 0 {
		if !isRepoAllowed(release) {
			log.Printf("ERROR: Server dis-allow this Repo.\n")
			c.JSON(http.StatusForbidden, gin.H{"status": "disallow release the software"})
			return
		}
	}

	err = DownloadTarballs(release)
	if err != nil {
		log.Printf("Can not download tarballs: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "can't download tarballs."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "release success"})
}

func main() {
	if config.Web.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if len(config.Web.Log) > 0 {
		gin.DisableConsoleColor()
		f, _ := os.Create(config.Web.Log)
		gin.DefaultWriter = io.MultiWriter(f)

	}
	router := gin.Default()
	router.POST("/release", PostRelease)
	router.StaticFS("/draft/", http.Dir(config.Path.Draft))
	router.StaticFS("/prerelease/", http.Dir(config.Path.PreRelease))
	router.StaticFS("/release/", http.Dir(config.Path.Release))
	router.StaticFS("/sources/", http.Dir(config.Path.Source))

	http.ListenAndServe(":"+strconv.Itoa(config.Web.Port), router)
}
