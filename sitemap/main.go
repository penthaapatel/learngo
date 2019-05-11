package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type Link struct {
	Href string
	Text string
}

func parse(r io.Reader) ([]Link, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	nodes := linkNodes(doc)
	var links []Link
	for _, node := range nodes {
		links = append(links, buildLink(node))
	}
	return links, nil
}

func buildLink(n *html.Node) Link {
	var ret Link
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			ret.Href = attr.Val
			break
		}
	}
	ret.Text = text(n)
	return ret
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type != html.ElementNode {
		return ""
	}
	var ret string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret += text(c)
	}
	return strings.Join(strings.Fields(ret), " ")
}

func linkNodes(n *html.Node) []*html.Node {
	if n.Type == html.ElementNode && n.Data == "a" {
		return []*html.Node{n}
	}
	var ret []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret = append(ret, linkNodes(c)...)
	}
	return ret
}

const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

type loc struct {
	Value string `xml:"loc"`
}

type urlset struct {
	Urls  []loc  `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

func main() {
	urlFlag := flag.String("url", "https://gophercises.com/", "url to bulild the sitemap for")
	depthFlag := flag.Int("depth", 3, "Depth to crawl the website")
	flag.Parse()
	//fmt.Printf("\n%+v\n", *reqUrl)

	//hrefs := getLinks(*urlFlag)

	hrefs := bfs(*urlFlag, *depthFlag)
	/* 	for _, href := range hrefs {
		fmt.Println(href)
	} */

	toXml := urlset{
		Xmlns: xmlns,
	}

	for _, href := range hrefs {
		toXml.Urls = append(toXml.Urls, loc{href})
	}

	fmt.Println(xml.Header)
	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("", "  ")
	if err := enc.Encode(toXml); err != nil {
		panic(err)
	}

	fmt.Println()

}

func getLinks(urlFlag string) []string {
	resp, err := http.Get(urlFlag)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	//io.Copy(os.Stdout, resp.Body)
	reqURL := resp.Request.URL
	baseURL := &url.URL{
		Scheme: reqURL.Scheme,
		Host:   reqURL.Host,
	}
	base := baseURL.String()
	//fmt.Println("Request URL: " + reqURL.String())
	//fmt.Println("Base URL:" + baseURL.String())
	return filter(hrefs(resp.Body, base), base)
}

func hrefs(r io.Reader, base string) []string {
	var ret []string
	links, _ := parse(r)
	for _, l := range links {
		switch {
		case strings.HasPrefix(l.Href, "/"):
			ret = append(ret, base+l.Href)
		case strings.HasPrefix(l.Href, "http"):
			ret = append(ret, l.Href)
		}
	}
	return ret
}

func filter(hrefs []string, base string) []string {
	var ret []string
	for _, l := range hrefs {
		switch {
		case strings.HasPrefix(l, base):
			ret = append(ret, l)
		}
	}
	return ret
}

type empty struct{}

func bfs(urlLink string, depth int) []string {
	seen := make(map[string]empty)
	var q map[string]empty
	nq := map[string]empty{
		urlLink: empty{},
	}

	for i := 0; i <= depth; i++ {
		q, nq = nq, make(map[string]empty)
		if len(q) == 0 {
			break
		}
		for urlStr, _ := range q {
			if _, ok := seen[urlStr]; ok {
				continue
			}

			seen[urlStr] = empty{}

			for _, link := range getLinks(urlStr) {
				if _, ok := seen[link]; !ok {
					nq[link] = empty{}
				}
			}
		}
	}

	ret := make([]string, 0, len(seen))
	for key, _ := range seen {
		ret = append(ret, key)
	}
	return ret

}
