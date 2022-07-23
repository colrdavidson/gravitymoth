package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"strings"
	"sort"
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
const rss_time_fmt = "Mon, 02 Jan 2006 15:04:05 -0700"

func generate_redirect(bin_name string, to string) {
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	redirect_str := fmt.Sprintf("<meta http-equiv=\"refresh\" content=\"0; URL=%s\" />", to)
	f.WriteString(redirect_str)
}

func generate_link_row(entries []BlogEntry, idx int, f *os.File) {
	if idx > 0 {
		prev_entry := entries[idx-1]
		prev_str := fmt.Sprintf("<a class=\"newer-link\" href=\"%s.html\"><i class=\"fa fa-arrow-left\"></i>Newer</a>", prev_entry.Slug)
		f.WriteString(prev_str)
	}
	if idx < len(entries)-1 {
		next_entry := entries[idx+1]
		next_str := fmt.Sprintf("<a class=\"older-link\" href=\"%s.html\">Older<i class=\"fa fa-arrow-right\"/></i></a>", next_entry.Slug)
		f.WriteString(next_str)
	}
}

func generate_posts(entries []BlogEntry, html_template string) {
	file_chunks := strings.Split(html_template, "{{header}}")
	if len(file_chunks) != 2 {
		log.Fatal("Template had no {{header}} tag!\n")
	}

	body_chunks := strings.Split(file_chunks[1], "{{content}}")
	if len(body_chunks) != 2 {
		log.Fatal("Template had no {{content}} tag!\n")
	}

	slug_chunks := strings.Split(body_chunks[1], "{{slugs}}")
	if len(slug_chunks) != 2 {
		log.Fatal("Template had no {{slugs}} tag!\n")
	}

	nav_foot_chunks := strings.Split(slug_chunks[1], "{{nav-foot}}")
	if len(nav_foot_chunks) != 2 {
		log.Fatal("Template had no {{nav-foot}} tag!\n")
	}

	chunks := []string{file_chunks[0], body_chunks[0], slug_chunks[0], nav_foot_chunks[0], nav_foot_chunks[1]}
	slugs := make([]string, 0)
	for _, md := range entries {
		slug_link := fmt.Sprintf("%s.html", md.Slug)
		li_str := fmt.Sprintf("<a class=\"slug-entry\" href=\"%s\"><li><p>%s</p></li></a>", slug_link, md.Title)
		slugs = append(slugs, li_str)
	}

	for i, entry := range entries {
		bin_name := fmt.Sprintf("%s%s.html", bin_dir, entry.Slug)
		if i == 0 {
			slug_link := fmt.Sprintf("%s.html", entry.Slug)
			headname := fmt.Sprintf("%sindex.html", bin_dir)
			generate_redirect(headname, slug_link)
		}

		f, err := os.Create(bin_name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		f.WriteString(chunks[0])

		date_str := entry.Date.Format(out_time_fmt)
		hdr_str := fmt.Sprintf("<h1>%s</h1><h5>%s</h5>", entry.Title, date_str)
		f.WriteString(hdr_str)

		f.WriteString("<div class=\"link-row\">")
		generate_link_row(entries, i, f)
		f.WriteString("</div>")

		f.WriteString(chunks[1])

		html := markdown.ToHTML(entry.Content, nil, nil)
		f.WriteString(string(html))
		f.WriteString(chunks[2])

		for j, s := range slugs {
			if j == i {
				li_str := fmt.Sprintf("<a class=\"slug-entry selected\"><li><p>%s</p></li></a>", entry.Title)

				f.WriteString(li_str)
			} else {
				f.WriteString(s)
			}
		}
		f.WriteString(chunks[3])

		generate_link_row(entries, i, f)

		f.WriteString(chunks[4])
	}
}

func generate_rss(entries []BlogEntry, rss_template string) {
	file_chunks := strings.Split(rss_template, "{{entries}}")
	if len(file_chunks) != 2 {
		log.Fatal("Template had no {{entries}} tag!\n")
	}
	chunks := []string{file_chunks[0], file_chunks[1]}

	bin_name := fmt.Sprintf("%sfeed.xml", bin_dir)
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.WriteString(chunks[0])

	for i, entry := range entries {
		date_str := entry.Date.Format(rss_time_fmt)	
		post_link := fmt.Sprintf("https://gravitymoth.com/blog/%s", entry.Slug)
		post_str := fmt.Sprintf("<item>\n\t\t<title>%s</title>\n\t\t<link>%s</link>\n\t\t<pubDate>%s</pubDate>\n\t</item>", entry.Title, post_link, date_str)

		if i != 0 {
			f.WriteString("\n\t")
		}

		f.WriteString(post_str)
	}

	f.WriteString(chunks[1])
}

func main() {
	files, err := ioutil.ReadDir(static_dir)
	if err != nil {
		log.Fatal(err)
	}

	var html_template_file fs.FileInfo = nil
	var rss_template_file fs.FileInfo = nil
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
		} else if file.Name() == "template.html" {
			html_template_file = file
		} else if file.Name() == "rss_template.xml" {
			rss_template_file = file
		}
	}

	if html_template_file == nil {
		log.Fatal("Couln't find html template!\n")
	}
	if rss_template_file == nil {
		log.Fatal("Couln't find rss template!\n")
	}

	_ = os.RemoveAll(bin_dir)
	_ = os.Mkdir(bin_dir, os.ModePerm)

	html_templpath := fmt.Sprintf("%s%s", static_dir, html_template_file.Name())
	html_template, err := os.ReadFile(html_templpath)
	if err != nil {
		log.Fatal(err)
	}

	rss_templpath := fmt.Sprintf("%s%s", static_dir, rss_template_file.Name())
	rss_template, err := os.ReadFile(rss_templpath)
	if err != nil {
		log.Fatal(err)
	}

	sort.SliceStable(mds, func(i, j int) bool {
		return mds[i].Date.After(mds[j].Date)
	})

	generate_posts(mds, string(html_template))
	generate_rss(mds,   string(rss_template))

	fmt.Printf("site generated\n")
}
