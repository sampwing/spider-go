package main

import ("fmt"
        "net/http"
		"regexp"
        "flag")

func readRedirect(req *http.Request, via []*http.Request) error {
	fmt.Println("upcomming req:", req)
        for idx := 0; idx < len(via); idx++ {
		fmt.Println("previous req:", idx, via[idx])
	}
	return nil
}

type URLMeta struct {
	extension string
	URL string
}

func URLMetaPrinter(um <-chan URLMeta, depth int) {
	for m := range um {
		if m.extension != "" {
			fmt.Println("depth:", depth, "URL:", m.URL, "extension:", m.extension)
		} else {
			fmt.Println("depth:", depth, "URL:", m.URL)
		}
		// recursively crawl found urls
		crawl(m.URL, depth - 1)
	}
}

func getURLMeta(urls <-chan []byte) <-chan URLMeta {
	out := make(chan URLMeta)
	rx, err := regexp.Compile("(?:\\.)[[:alpha:]]+$")
	if err != nil {
		return out
	}
	go func() {
		for url := range urls {
			// for each url try to grab the extension
			match := rx.Find(url)
			mt  := URLMeta{URL: string(url[:]), extension: string(match[:])}
			out <- mt
		}
		close(out)
	}()
	return out
}

func findUrls(bytes []byte, depth int) {
	urlRx, err := regexp.Compile("https?://[[:alpha:]]+([.]?[[:alpha:]]+)*?(/+[[:alpha:]]+)+(/|(\\.[[:alpha:]]+))?")
	if err != nil {
		return
	}
	urls := urlRx.FindAll(bytes, -1)
	urlCount := len(urls)

	// create a channel to send all of the urls we encounter
	urlChan := make(chan []byte, 10000)
	for idx := 0; idx < urlCount; idx++ {
		urlChan <- urls[idx]
	}
	close(urlChan)
	urlmeta := getURLMeta(urlChan)
	URLMetaPrinter(urlmeta, depth)

}

func crawl(host string, depth int) {
	if depth < 0 {
		// only crawl until depth has reached the bottom or zero
		return
	}
	client := &http.Client{
		CheckRedirect: readRedirect,
	}
	body := make([]byte, 100000000)
	req, _ := http.NewRequest("GET", host, nil)
	req.Header.Set("User-Agent", "Crawler")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("hit error when opening", host)
		fmt.Println(err)
	}
	defer resp.Body.Close()

	if resp != nil {
		//fmt.Println("status:", resp.Status)
		for {
			_, err := resp.Body.Read(body)
			if err != nil {
				return
			}
			findUrls(body[:], depth)
		}
	}
}

func main(){
	var host string
	flag.StringVar(&host, "host", "http://www.reddit.com", "host to start crawling from")
	flag.Parse()
	crawl(host, 0)
}
