package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	. "github.com/jaytaylor/html2text"
)

// Set your email here to include in the User-Agent string.
var email = "youremail@gmail.com"
var urls = []string{
	//"http://techcrunch.com/",
	//"https://www.reddit.com/",
	"https://en.wikipedia.org/wiki/Main_Page",
	//"https://news.ycombinator.com/",
	//"https://www.buzzfeed.com/",
	"http://digg.com",
	"https://dou.ua",
}

func respGen(urls ...string) <-chan *http.Response {
	var wg sync.WaitGroup
	out := make(chan *http.Response)
	wg.Add(len(urls))
	for _, url := range urls {
		go func(url string) {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				panic(err)
			}
			req.Header.Set("user-agent", "testBot("+email+")")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			out <- resp
			wg.Done()
		}(url)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func rootGen(in <-chan *http.Response) <-chan *html.Node {
	var wg sync.WaitGroup
	out := make(chan *html.Node)
	for resp := range in {
		wg.Add(1)
		go func(resp *http.Response) {
			root, err := html.Parse(resp.Body)
			if err != nil {
				panic(err)
			}
			out <- root
			wg.Done()
		}(resp)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func contentGen(in <-chan *html.Node) <-chan map[string]string {
	// define a matcher
	//matcher := func(n *html.Node) bool {
	//	// must check for nil values
	//	if n.DataAtom == atom.A && n.Parent != nil && n.Parent.Parent != nil {
	//		return scrape.Attr(n.Parent.Parent, "class") == "athing"
	//	}
	//	return false
	//}
	var wg sync.WaitGroup
	out := make(chan map[string]string)
	for root := range in {
		wg.Add(1)
		go func(root *html.Node) {
			//my_html, ok := scrape.Find(root, scrape.ByTag(atom.Html))
			//titles := scrape.FindAllNested(root, matcher)
			htmls := scrape.FindAllNested(root, scrape.ByTag(atom.Body))
			for _, my_html := range htmls{
				//if ok {
				// out <- scrape.Text(title)
				text, err := FromString(scrape.Text(my_html), Options{PrettyTables: true})
				if err != nil {
					panic(err)
				}
				content := map[string]string{
					text : text,
				}
				out <- content
			}
			//for _, my_html := range htmls{
			//	text, err := FromString(scrape.Text(my_html), Options{PrettyTables: true})
			//	if err != nil {
			//		panic(err)
			//	}
			//	//out <- scrape.Text(my_html)
			//	out <- text
			//}

			wg.Done()
		}(root)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {


	// Set up the pipeline to consume back-to-back output
	// ending with the final stage to print the content of
	// each web page in the main go routine.gi
	for content := range contentGen(rootGen(respGen(urls...))) {
		fmt.Println(content)

	}


}
