package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/a8m/mark"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

type RssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel channel  `xml:"channel"`
}

type channel struct {
	Title         string `xml:"title"`
	Description   string `xml:"description"`
	Link          string `xml:"link"`
	LastBuildDate string `xml:"lastBuildDate"`
	PubDate       string `xml:"pubDate"`
	Items         Items  `xml:"item"`
}

type item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Guid        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

type Items []item

func (p Items) Len() int { return len(p) }

func (p Items) Less(i, j int) bool {
	var a, b time.Time
	var err error

	a, err = time.Parse(time.RFC1123Z, p[i].PubDate)
	if err != nil {
		return false
	}
	b, err = time.Parse(time.RFC1123Z, p[j].PubDate)
	if err != nil {
		return false
	}
	// Here use After, not Before, so we didn't need to reverse them.
	return a.After(b)
}
func (p Items) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func loadCurrentFeed() (feed RssFeed, err error) {
	var file *os.File
	var data []byte

	file, err = os.Open(config.Rss.Path)
	if err != nil {
		return
	}
	defer file.Close()

	data, err = ioutil.ReadAll(file)
	if err != nil {
		return
	}
	err = xml.Unmarshal(data, &feed)
	if err != nil {
		log.Printf("error:%v", err)
		return
	}
	return
}

func getUrlPrefix(release Release) string {
	memOut := new(bytes.Buffer)

	tmpl, err := template.New("urlPrefix").Parse(config.Rss.UrlPrefix)
	if err != nil {
		log.Printf("parsing: %s", err)
		return ""
	}

	err = tmpl.Execute(memOut, release)
	if err != nil {
		log.Printf("parsing: %s", err)
		return ""
	}
	return memOut.String()
}

func genNewItem(release Release) (new item) {
	new.Title = release.Name + " " + release.Version

	url := getUrlPrefix(release)

	news := strings.Split(release.News, "\n")
	news = append(news, "---")
	news = append(news, fmt.Sprintf("[News](%s/%s-%s.news)", url, release.Name, release.Version))
	news = append(news, "\nDownload")
	for _, file := range release.Files {
		news = append(news, fmt.Sprintf("- [%s](%s/%s)", file.Name, url, file.Name))
	}

	for _, file := range release.Files {
		news = append(news, fmt.Sprintf("- [%s](%s) (From Github)", file.Name, file.Url))
	}

	new.Description = mark.Render(strings.Join(news, "\n") + "\n")
	new.Link = fmt.Sprintf("%s/%s-%s.tar.xz", url, release.Name, release.Version)
	new.PubDate = release.PublishedAt.Format(time.RFC1123Z)
	new.Guid = release.Name + "-" + release.Version
	return
}

func updateFeed(release Release) {
	feed, err := loadCurrentFeed()
	if err != nil {
		return
	}

	newItem := genNewItem(release)

	feed.Channel.Items = append(feed.Channel.Items, newItem)

	sort.Sort(feed.Channel.Items)
	if len(feed.Channel.Items) > config.Rss.Count {
		feed.Channel.Items = feed.Channel.Items[:config.Rss.Count]
	}

	feed.Channel.PubDate = newItem.PubDate
	feed.Channel.LastBuildDate = time.Now().Format(time.RFC1123Z)
	if config.Rss.Title != "" {
		feed.Channel.Title = config.Rss.Title
	}
	if config.Rss.Description != "" {
		feed.Channel.Description = config.Rss.Description
	}
	if config.Rss.Link != "" {
		feed.Channel.Link = config.Rss.Link
	}

	output, err := xml.MarshalIndent(feed, " ", "  ")
	if err != nil {
		log.Printf("error:%v\n", err)
	}

	out, err := os.Create(config.Rss.Path)
	if err != nil {
		log.Printf("error:%v\n", err)
	}
	out.Write([]byte(xml.Header))
	out.Write(output)
	out.Close()
}
