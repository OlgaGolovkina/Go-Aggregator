package main

import (
	"fmt"
	"net/http"
	"sync"
	"database/sql"
	"strconv"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

// Set your email here to include in the User-Agent string.
//var email = "youremail@gmail.com"
var urls = []string{
	//"http://techcrunch.com/",
	//"https://www.reddit.com/",
	//"https://en.wikipedia.org",
	"https://news.ycombinator.com/",
	"https://www.buzzfeed.com/",
	"https://dou.ua",
	"http://digg.com",
}
var keyword  = "Python"
var id int

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
			//req.Header.Set("user-agent", "testBot("+email+")")
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

func titleGen(in <-chan *html.Node) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)
	for root := range in {
		wg.Add(1)
		go func(root *html.Node) {
			// define a matcher
			matcher := func(n *html.Node) bool {
				// must check for nil values
				if n.DataAtom == atom.A && n.Parent != nil && n.Parent.Parent != nil {
					return scrape.Attr(n.Parent.Parent, "class") == "athing"
				}
				return false
			}

			titles := scrape.FindAllNested(root, matcher)
			for _, title := range titles {
				out <- scrape.Text(title)
			}
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
	// ending with the final stage to print the title of
	// each web page in the main go routine.
	for title := range titleGen(rootGen(respGen(urls...))) {
		a := strings.Contains(title, keyword)
		if  a == true {
			database, _ := sql.Open("sqlite3", "./titles.db")
			statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS titles (id INTEGER PRIMARY KEY, title TEXT)")
			statement.Exec()
			statement, _ = database.Prepare("INSERT INTO titles (title) VALUES (?)")
			statement.Exec(title)
			rows, _ := database.Query("SELECT id, title FROM titles")

			for rows.Next() {
				rows.Scan(&id, &title)
				fmt.Println(strconv.Itoa(id) + ": " + title + " ")
			}
			statement, _ = database.Prepare("DROP TABLE [IF EXISTS] titles")
		}
	}
	fmt.Println("\nThere are no useful articles for you")
}