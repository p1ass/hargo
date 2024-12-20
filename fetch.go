package hargo

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Fetch downloads all resources references in .har file
func Fetch(r *bufio.Reader, outdir string) error {
	har, err := Decode(r)

	check(err)

	datestring := time.Now().Format("20060102150405")
	if outdir == "" {
		outdir = "." + string(filepath.Separator) + "hargo-fetch-" + datestring
	}

	err = os.Mkdir(outdir, 0777)

	check(err)

	jar, _ := cookiejar.New(nil)

	for i, entry := range har.Log.Entries {

		// TODO create goroutine here to parallelize requests

		fmt.Println("URL: " + entry.Request.URL)

		var req *http.Request
		if entry.Request.Method != http.MethodGet {
			req, _ = http.NewRequest(entry.Request.Method, entry.Request.URL, strings.NewReader(entry.Request.PostData.Text))
		} else {
			req, _ = http.NewRequest(entry.Request.Method, entry.Request.URL, nil)
		}

		for _, h := range entry.Request.Headers {
			if !strings.HasPrefix(h.Name, ":") && h.Name != "Cookie" {
				req.Header.Add(h.Name, h.Value)
			}
		}

		// for _, c := range entry.Request.Cookies {
		// 	cookie := &http.Cookie{Name: c.Name, Value: c.Value, HttpOnly: false, Domain: c.Domain}
		// 	req.AddCookie(cookie)
		// }

		// cookie := &http.Cookie{Name: "_hargo", Value: "true", HttpOnly: false}
		// req.AddCookie(cookie)

		isJSON := entry.Response.Content.MimeType == "application/json"
		err = downloadFile(req, outdir, i, isJSON, jar)

		if err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

func downloadFile(req *http.Request, outdir string, num int, isJSON bool, jar http.CookieJar) error {

	fileName := path.Base(req.URL.Path)

	if fileName == "/" || fileName == "" {
		fileName = "index.html"
	}

	fileName = filepath.Join(outdir, strings.ReplaceAll(filepath.Join(fmt.Sprintf("%03d", num), filepath.Dir(req.URL.Path), fileName), "/", "_"))

	if isJSON {
		fileName += ".json"
	}

	if len(fileName) == 0 {
		return nil
	}

	file, err := os.Create(fileName)

	if err != nil {
		log.Error(err)
		return err
	}
	defer file.Close()

	// jar.SetCookies(req.URL, req.Cookies())

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
		Jar:       jar,
		Transport: http.DefaultTransport,
	}

	// spew.Dump(client)
	// spew.Dump(req)

	resp, err := client.Do(req) // .Get(rawURL) // add a filter to check redirect

	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	if isJSON {
		var bs any
		if err := json.NewDecoder(resp.Body).Decode(&bs); err != nil {
			log.Error(err)
			return err
		}
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if err := enc.Encode(bs); err != nil {
			log.Error(err)
			return err
		}
	} else {

		size, err := io.Copy(file, resp.Body)
		if err != nil {
			log.Error(err)
			return err
		}
		fmt.Printf("Downloaded %s [%v bytes]\n", fileName, size)
	}

	return nil
}
