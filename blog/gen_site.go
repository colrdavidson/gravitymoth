package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"time"
	"strings"
	"sort"
	"flag"
	"io/ioutil"
	"io/fs"
	"github.com/gomarkdown/markdown"
)

type BlogEntry struct {
	Name string
	Path string
	Content []byte

	Title string
	Date time.Time
	Slug string
}

const static_dir = "static/"
const bin_dir = "docs/"
const in_time_fmt = "01-02-2006 15:04 MST"
const out_time_fmt = "Mon, 02 Jan 2006"

func generate_redirect(bin_name string, to string) {
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	redirect_str := fmt.Sprintf("<meta http-equiv=\"refresh\" content=\"0; URL=%s\" />", to)
	f.WriteString(redirect_str)
}

func generate_file(entry BlogEntry, chunks []string, slugs []string, sub_dir string, bin_name string) {
}

func main() {
	local_ptr := flag.Bool("local", false, "uses local dir instead of /blog")
	flag.Parse()

	sub_dir := "/blog"
	if *local_ptr {
		sub_dir = ""
	}

	files, err := ioutil.ReadDir(static_dir)
	if err != nil {
		log.Fatal(err)
	}

	var template_file fs.FileInfo = nil
	var style_file fs.FileInfo = nil
	mds := make([]BlogEntry, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".blg") {
			staticname := fmt.Sprintf("%s%s", static_dir, file.Name())
			content, err := os.ReadFile(staticname)
			if err != nil {
				log.Fatal(err)
			}

			title := ""
			date_str := ""
			slug := ""
			keymap := make(map[string]bool)
			keymap["title: "] = false
			keymap["date: "] = false
			keymap["slug: "] = false

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				for k, _ := range keymap {
					if strings.HasPrefix(line, k) {
						val := line[len(k):]
						switch k {
						case "title: ":
							title = val
						case "date: ":
							date_str = val
						case "slug: ":
							slug = val
						}
					}
				}
			}
			content = content[strings.Index(string(content), "\n\n")+1:]

			date, err := time.Parse(in_time_fmt, string(date_str))
			if err != nil {
				log.Fatal(err)
			}

			md := BlogEntry{Name: file.Name(), Path: static_dir, Content:
content, Title: string(title), Date: date, Slug: string(slug)}
			mds = append(mds, md)
		} else if strings.HasSuffix(file.Name(), ".html") {
			template_file = file
		} else if strings.HasSuffix(file.Name(), ".css") {
			style_file = file
		}
	}

	if template_file == nil {
		log.Fatal("Couln't find template host!\n")
	}

	_ = os.RemoveAll(bin_dir)
	_ = os.Mkdir(bin_dir, os.ModePerm)

	style_in_path := fmt.Sprintf("%s%s", static_dir, style_file.Name())
	style_out_path := fmt.Sprintf("%s%s", bin_dir, style_file.Name())
	style_in, err := os.Open(style_in_path)
	if err != nil {
		log.Fatal(err)
	}
	style_out, err := os.Create(style_out_path)
	_, err = io.Copy(style_out, style_in)
	if err != nil {
		log.Fatal(err)
	}

	templpath := fmt.Sprintf("%s%s", static_dir, template_file.Name())
	template, err := os.ReadFile(templpath)
	if err != nil {
		log.Fatal(err)
	}

	file_chunks := strings.Split(string(template), "{{content}}")
	if len(file_chunks) != 2 {
		log.Fatal("Template had no {{content}} tag!\n")
	}

	more_chunks := strings.Split(file_chunks[1], "{{slugs}}")
	if len(more_chunks) != 2 {
		log.Fatal("Template had no {{slugs}} tag!\n")
	}

	nav_foot_chunks := strings.Split(more_chunks[1], "{{nav-foot}}")
	if len(nav_foot_chunks) != 2 {
		log.Fatal("Template had no {{nav-foot}} tag!\n")
	}

	chunks := []string{file_chunks[0], more_chunks[0], nav_foot_chunks[0], nav_foot_chunks[1]}

	sort.SliceStable(mds, func(i, j int) bool {
		return mds[i].Date.After(mds[j].Date)
	})

	slugs := make([]string, len(mds))
	for _, md := range mds {
		slug_link := fmt.Sprintf("%s/%s.html", sub_dir, md.Slug)
		li_str := fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", slug_link, md.Title)
		slugs = append(slugs, li_str)
	}

	headname := fmt.Sprintf("%sindex.html", bin_dir)
	for i, entry := range mds {
		bin_name := fmt.Sprintf("%s%s.html", bin_dir, entry.Slug)
		if i == 0 {
			slug_link := fmt.Sprintf("%s/%s.html", sub_dir, entry.Slug)
			generate_redirect(headname, slug_link)
		}

		f, err := os.Create(bin_name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		html := markdown.ToHTML(entry.Content, nil, nil)

		f.WriteString(chunks[0])

		date_str := entry.Date.Format(out_time_fmt)
		hdr_str := fmt.Sprintf("<h1>%s</h1><h5>%s</h5>", entry.Title, date_str)
		f.WriteString(hdr_str)
		f.WriteString(string(html))
		f.WriteString(chunks[1])
		for _, s := range slugs {
			f.WriteString(s)
		}
		f.WriteString(chunks[2])

		if i > 0 {
			prev_entry := mds[i-1]
			prev_str := fmt.Sprintf("<a class=\"newer-link\" href=\"%s/%s.html\">Newer</a>", sub_dir, prev_entry.Slug)
			f.WriteString(prev_str)
		}
		if i < len(mds)-1 {
			next_entry := mds[i+1]
			next_str := fmt.Sprintf("<a class=\"older-link\" href=\"%s/%s.html\">Older</a>", sub_dir, next_entry.Slug)
			f.WriteString(next_str)
		}

		f.WriteString(chunks[3])
	}

	fmt.Printf("site generated\n")
}
