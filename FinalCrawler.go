package main

//import packages
import (
	"fmt"		//GO’s base package 
	"io/ioutil"	//reading/writing data from input/output streams 
	"net/http"	//for sending HTTP requests
	"net/url"	//for URL formatting
	"regexp"	//regular expressions
	"runtime"	//GO runtime (used to set the number of threads to be used)
	"strings"	//string manipulation and testing
)

//how many threads to use within the application
const NCPU = 8

//URL filter function definition
type filterFunc func(string, Crawler) bool

//Our crawler structure definition
type Crawler struct {
	//the base URL of the website being crawled
	host string
	//a channel on which the crawler will receive new (unfiltered) URLs to crawl
	//the crawler will pass everything received from this channel
	//through the chain of filters we have
	//and only allowed URLs will be passed to the filteredUrls channel
	urls chan string
	//a channel on which the crawler will receive filtered URLs.
	filteredUrls chan string //a channel
	//a slice that contains the filters we want to apply on the URLs.
	filters []filterFunc
	//a regular expression pointer to the RegExp that will be used to extract the
	//URLs from each request.
	re *regexp.Regexp
	//an integer to track how many URLs have been crawled
	count int
}

//starts the crawler
//the method starts two GO functions
//the first one waits for new URLs as they
//get extracted.
//the second waits for filtered URLs as they
//pass through all the registered filters
func (crawler *Crawler) start() {
	//wait for new URLs to be extracted and passed to the URLs channel.
	go func() {
		for n := range crawler.urls {
			//filter the url
			go crawler.filter(n)
		}
	}()

	//wait for filtered URLs to arrive through the filteredUrls channel
	go func() {
		for s := range crawler.filteredUrls {
			//print the newly received filtered URL
			fmt.Println(s)
			//increment the crawl count
			crawler.count++
			//print crawl count
			fmt.Println(crawler.count)
			//start a new GO routine to crawl the filtered URL
			go crawler.crawl(s)
		}
	}()
}

//given a URL, the method will send an HTTP GET request
//extract the response body
//extract the URLs from the body
func (crawler *Crawler) crawl(url string) {
	//send http request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("An error has occured")
		fmt.Println(err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Read error has occured")
		} else {
			strBody := string(body)
			crawler.extractUrls(url, strBody)
		}

	}
}

//adds a new URL filter to the crawler
func (crawler *Crawler) addFilter(filter filterFunc) Crawler {
	crawler.filters = append(crawler.filters, filter)
	return crawler
}

//stops the crawler by closing both the URLs channel
//and the filtered URLs channel
func (crawler *Crawler) stop() {
	close(crawler.urls)
	close(crawler.filteredUrls)
}

//given a URL, the method will apply all the filters
//on that URL, if and only if, it passes through all
//the filters, it will then be passed to the filteredUrls channel
func (crawler *Crawler) filter(url string) {
	temp := false
	for _, fn := range crawler.filters {
		temp = fn(url, crawler)
		if temp != true {
			return
		}
	}
	crawler.filteredUrls <- url
}

//given the crawled URL, and its body, the method
//will extract the URLs from the body
//and generate absolute URLs to be crawled by the
//crawler
//the extracted URLs will be passed to the URLs channel
func (crawler *Crawler) extractUrls(Url, body string) {
	newUrls := crawler.re.FindAllStringSubmatch(body, -1)
	u := ""
	baseUrl, _ := url.Parse(Url)
	if newUrls != nil {
		for _, z := range newUrls {
			u = z[1]
			ur, err := url.Parse(z[1])
			if err == nil {
				if ur.IsAbs() == true {
					crawler.urls <- u
				} else if ur.IsAbs() == false {
					crawler.urls <- baseUrl.ResolveReference(ur).String()
				} else if strings.HasPrefix(u, "//") {
					crawler.urls <- "http:" + u
				} else if strings.HasPrefix(u, "/") {
					crawler.urls <- crawler.host + u
				} else {
					crawler.urls <- Url + u
				}
			}
		}
	}
}

func main() {
	//set how many processes (threads to use)
	runtime.GOMAXPROCS(NCPU)

	//create a new instance of the crawler structure
	c := Crawler{
		"http://www.thesaurus.com/",
		make(chan string),
		make(chan string),
		make([]filterFunc, 0),
		regexp.MustCompile("(?s)<a[ t]+.*?href="(http.*?)".*?>.*?</a>"),
		0,
	}
	//add our only filter which makes sure that we are only
	//crawling internal URLs.
	c.addFilter(func(Url string, crawler Crawler) bool {
		return strings.Contains(Url, crawler.host)
	}).start()

	c.urls <- c.host

	var input string
	fmt.Scanln(&input)
