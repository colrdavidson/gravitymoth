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
	Description string
	Thumbnail string
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
	chunks := make([]string, 0)

	// these have to be in descending occuring order
	tags := []string{"{{description}}", "{{unfurl}}", "{{header}}", "{{slug}}", "{{content}}", "{{slugs}}", "{{nav-foot}}"}

	cur_chunk := html_template
	for i, tag := range tags {
		tmp := strings.Split(cur_chunk, tag)
		if len(tmp) != 2 {
			log.Fatalf("Template had no %s tag!\n", tag)
		}

		if i == (len(tags) - 1) {
			chunks = append(chunks, tmp...)
			continue
		}

		chunks = append(chunks, tmp[0])
		cur_chunk = tmp[1]
	}

	slugs := make([]string, 0)
	for _, md := range entries {
		slug_link := fmt.Sprintf("%s.html", md.Slug)
		li_str := fmt.Sprintf(`<a class="slug-entry" href="%s"><li><p>%s</p></li></a>`, slug_link, md.Title)
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

		// Generate description
		{
			f.WriteString(entry.Description)
		}

		f.WriteString(chunks[1])

		// Generate unfurl
		{
			f.WriteString(`<meta property="og:type"  content="article" />`)

			url_str := fmt.Sprintf(`<meta property="og:url" content="https://gravitymoth.com/blog/%s" />`, entry.Slug)
			f.WriteString(url_str)

			desc_str := fmt.Sprintf(`<meta property="og:description" content="%s" />`, entry.Description)
			f.WriteString(desc_str)

			title_str := fmt.Sprintf(`<meta property="og:title" content="%s" />`, entry.Title)
			twitter_title_str := fmt.Sprintf(`<meta property="twitter:title" content="%s" />`, entry.Title)
			f.WriteString(title_str)
			f.WriteString(twitter_title_str)

			image_str := fmt.Sprintf(`<meta property="og:image" content="https://gravitymoth.com/media/%s" />`, entry.Thumbnail)
			f.WriteString(image_str)

			f.WriteString(`<meta property="og:locale" content="en_US">`)
			f.WriteString(`<meta property="og:site_name" content="Gravity Moth">`)
			f.WriteString(`<meta property="twitter:card" content="summary">`)
		}

		f.WriteString(chunks[2])

		date_str := entry.Date.Format(out_time_fmt)
		hdr_str := fmt.Sprintf("<h1>%s</h1><h5>%s</h5>", entry.Title, date_str)
		f.WriteString(hdr_str)

		f.WriteString(`<div class="link-row">`)
		generate_link_row(entries, i, f)
		f.WriteString(`</div>`)
		f.WriteString(chunks[3])

		slug_class := fmt.Sprintf("slug-%s", entry.Slug)
		f.WriteString(slug_class)
		f.WriteString(chunks[4])

		html := markdown.ToHTML(entry.Content, nil, nil)
		f.WriteString(string(html))
		f.WriteString(chunks[5])

		for j, s := range slugs {
			if j == i {
				li_str := fmt.Sprintf(`<a class="slug-entry selected"><li><p>%s</p></li></a>`, entry.Title)

				f.WriteString(li_str)
			} else {
				f.WriteString(s)
			}
		}
		f.WriteString(chunks[6])

		generate_link_row(entries, i, f)

		f.WriteString(chunks[7])
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
		post_str := fmt.Sprintf(`<item><title>%s</title><link>%s</link><description>%s</description><pubDate>%s</pubDate><guid isPermaLink="true">%s</guid></item>`, entry.Title, post_link, entry.Description, date_str, post_link)

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
			desc := ""
			img := ""

			keymap := make(map[string]bool)
			keymap["title: "] = false
			keymap["date: "] = false
			keymap["desc: "] = false
			keymap["slug: "] = false
			keymap["img: "] = false

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
						case "desc: ":
							desc = val
						case "img: ":
							img = val
						}
					}
				}
			}
			content = content[strings.Index(string(content), "\n\n")+1:]

			date, err := time.Parse(in_time_fmt, string(date_str))
			if err != nil {
				log.Fatal(err)
			}

			// If the post doesn't have a thumbnail, use the default site logo instead
			if img == "" {
				img = "logo_bg.png"
			}

			md := BlogEntry{Name: file.Name(), Path: static_dir, Content:
content, Title: string(title), Date: date, Slug: string(slug), Description: string(desc), Thumbnail: string(img)}
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
