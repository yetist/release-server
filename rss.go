package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
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
	//return a.Before(b)
	return a.After(b)
}
func (p Items) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func generateXML() {
	var items []item
	now := time.Now()
	v := &RssFeed{Version: "2.0"}
	c := channel{"MATE releases", "RSS feed for MATE releases", "http://pub.mate-desktop.org/rss.xml", now.Format(time.RFC1123Z), now.Format(time.RFC1123Z), items}
	c.Items = append(c.Items, item{"mate-common", "released by MATE Desktop Team", "http://abc.com", "mate-common", now.Format(time.RFC1123Z)})
	c.Items = append(c.Items, item{"mate-common2", "released by MATE Desktop Team", "http://abc.com", "mate-common2", now.Format(time.RFC1123Z)})
	v.Channel = c
	output, err := xml.MarshalIndent(v, " ", "  ")
	if err != nil {
		fmt.Println("error:%v\n", err)
	}
	os.Stdout.Write([]byte(xml.Header))
	os.Stdout.Write(output)
}

func readCurrentFeed() (feed RssFeed, err error) {
	var file *os.File
	var data []byte

	file, err = os.Open(config.Path.Rss)
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
		fmt.Println("error:%v", err)
		return
	}
	return
}

func updateFeed(release Release) {
	feed, err := readCurrentFeed()
	if err != nil {
		return
	}
	//fmt.Printf("%#v\n", feed)

	var newItem item
	newItem.Title = release.Name + " " + release.Version
	newItem.Description = release.News
	newItem.Link = "https://"
	newItem.PubDate = release.PublishedAt.Format(time.RFC1123Z)

	feed.Channel.Items = append(feed.Channel.Items, newItem)
	feed.Channel.PubDate = newItem.PubDate

	for _, item := range feed.Channel.Items {
		fmt.Printf("%#v\n", item.Title)
	}

	sort.Sort(feed.Channel.Items)
	feed.Channel.Items = feed.Channel.Items[:3]
	feed.Channel.LastBuildDate = time.Now().Format(time.RFC1123Z)

	output, err := xml.MarshalIndent(feed, " ", "  ")
	if err != nil {
		fmt.Println("error:%v\n", err)
	}
	os.Stdout.Write([]byte(xml.Header))
	os.Stdout.Write(output)
}
