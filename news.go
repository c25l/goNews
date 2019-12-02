package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	feeds = []string{}
	loc = ""
	config = "config.json"
	newPage   = "news.html"
	oldPage   = "yesterday.html"
	savedPage = ""
	refresh = time.Minute
)
type Config struct {
	Feeds []string `json:"feeds"`
	Location string `json:"location"`
}

type Feed struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

func fetch(loc string) string {
	client := http.DefaultClient
	response, err := client.Get(loc)
	if err != nil {
		return "error 1"
	}
	defer response.Body.Close()
	xmlDecoder := xml.NewDecoder(response.Body)
	var rss struct {
		Feed Feed `xml:"channel"`
	}
	if err = xmlDecoder.Decode(&rss); err != nil {
		return "error 2"
	}
	var out strings.Builder
	out.WriteString("<hr><h1>")
	out.WriteString(rss.Feed.Title)
	out.WriteString("</h1>\n")
	trigger := false
	for _, xx := range rss.Feed.Items {
		itemDate, err := time.Parse(time.RFC1123Z, xx.PubDate)
		if err != nil {
			itemDt, err := time.Parse(time.RFC1123, xx.PubDate)
			if err != nil {
				log.Print(err)
			}
			itemDate = itemDt
		}
		if time.Now().Sub(itemDate) < 24*time.Hour {
			trigger = true
			out.WriteString("</div><h2>" + xx.Title + "</h2>\n<p>")
			out.WriteString(xx.Description)
			out.WriteString("</p> <a href=\"" + xx.Link + "\">go</a></div><br>\n")
		}

	}
	// confirm time proximity
	// render as html
	if trigger {
		return out.String()
	}
	return ""
}

type Weather struct {
	Properties Property `json:"properties"`
}
type Property struct {
	Periods []Period `json:"periods"`
}
type Period struct {
	Detailed string `json:"detailedForecast"`
}
func getConfig() { 
	confFile, err := os.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	var x Config
	jsonDecoder := json.NewDecoder(confFile)
	err = jsonDecoder.Decode(&x)
	if err != nil {
		log.Fatal(err)
	}
	feeds = x.Feeds
	loc = x.Location
}
func weather(loc string) string {
	client := http.DefaultClient
	response, err := client.Get("https://api.weather.gov/gridpoints/" + loc + "/forecast")
	if err != nil {
		return "error 0"
	}
	defer response.Body.Close()
	jsonDecoder := json.NewDecoder(response.Body)
	var x Weather
	if err = jsonDecoder.Decode(&x); err != nil {
		fmt.Print(err)
		return "error 1"
	}
	var out strings.Builder
	out.WriteString("<h1>Weather</h1>\n<h3>Today</h3><p>\n")
	out.WriteString(x.Properties.Periods[0].Detailed)
	out.WriteString("\n</p><h3>Tonight</h3><p>\n")
	out.WriteString(x.Properties.Periods[1].Detailed)
	out.WriteString("\n</p><h3>Tomorrow</h3><p>\n")
	out.WriteString(x.Properties.Periods[2].Detailed)
	out.WriteString("\n")
	return out.String()
}

func date() string {
	return ""
}

func genPage(newstuff, oldstuff string) string {
	var x strings.Builder
	x.WriteString(`
---
title: "`)
x.WriteString(todayString())
x.WriteString(`"
---
<html><title>Old News</title><body>`)
	x.WriteString(newstuff)
	x.WriteString(`<br><hr><br>`)
	x.WriteString(oldstuff)
	x.WriteString(`</body></html>`)
	return x.String()
}
func todayString() string{
	now := time.Now()
	return fmt.Sprintf("%02d%02d", now.Month(), now.Day())
}

func update() string {
	var out strings.Builder
	out.WriteString(weather(loc))
	for _, xx := range feeds {
		out.WriteString(fetch(xx))
	}

	return out.String()
}

func rebuildPage() {
	confFileInfo, err := os.Stat(config)
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	fileModTime := confFileInfo.ModTime()
	confRefresh := now.Sub(fileModTime) <= 2*refresh
	if confRefresh {
		getConfig()
	}
	newFileInfo, err := os.Stat(newPage)
	if err != nil {
		log.Fatal(err)
	}
	fileModTime = newFileInfo.ModTime()
	if (now.Sub(fileModTime) <= 2*refresh) && !confRefresh {
		return
	}
	newFile, err := os.Open(newPage)
	if err != nil {
		log.Fatal(err)
	}
	newData, err := ioutil.ReadAll(newFile)
	if err != nil {
		log.Fatal(err)
	}
	oldFile, err := os.Open(oldPage)
	if err != nil {
		log.Fatal(err)
	}
	oldData, err := ioutil.ReadAll(oldFile)
	if err != nil {
		log.Fatal(err)
	}
	savedPage = genPage(string(newData), string(oldData))
}

func rebuildPageSometimes() {
	rebuildPage()
	tick := time.NewTicker(refresh)
	for _ = range tick.C {
		rebuildPage()
	}
}

func main() {
	prefix := flag.String("prefix", "/home/pi/Sync/code/website/content/","path prefix to save content to.")
	conf := flag.String("config", "/home/pi/Sync/code/goNews/config.json", "location of config file")
	flag.Parse()

	config = *conf
	newPage = *prefix + newPage
	oldPage = *prefix + oldPage
	getConfig()

	newData := update()
	newFile, err := os.Open(newPage)
	if err != nil {
		newFile, err = os.Create(newPage)
		if err != nil {
			log.Fatal(err)
		}
	}
	oldData, err := ioutil.ReadAll(newFile)
	if err != nil {
		log.Fatal(err)
	}
	newFile.Close()
	err = ioutil.WriteFile(oldPage, oldData, 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(newPage, []byte(genPage(newData,"")), 0644)
	if err != nil {
		log.Fatal(err)
	}

}
