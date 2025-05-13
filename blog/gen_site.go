package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

type BlogEntry struct {
	Name    string
	Path    string
	Content []byte

	Title       string
	Date        time.Time
	Slug        string
	Description string
	Thumbnail   string
}

type PostTemplateData struct {
	Description string
	Unfurl      template.HTML
	Header      template.HTML
	NavHead     template.HTML
	Slug        string
	Content     template.HTML
	Slugs       template.HTML
	NavFoot     template.HTML
}

const static_dir = "static/"
const bin_dir = "docs/"
const in_time_fmt = "01-02-2006 15:04 MST"
const out_time_fmt = "Mon, 02 Jan 2006"
const rss_time_fmt = "Mon, 02 Jan 2006 15:04:05 -0700"

func generate_slug(e BlogEntry) string {
	return "blog/" + fmt.Sprintf("%s", e.Slug)
}

func generate_redirect(bin_name string, to string) {
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	redirect_str := fmt.Sprintf("<meta name=\"color-scheme\" content=\"light dark\"><meta http-equiv=\"refresh\" content=\"0; URL=%s\" />", to)
	f.WriteString(redirect_str)
}

func generate_link_row(entries []BlogEntry, idx int) template.HTML {
	linkRowStr := ""
	if idx > 0 {
		prev_entry := entries[idx-1]
		prev_str := fmt.Sprintf("<a class=\"newer-link\" href=\"%s\"><i class=\"fa fa-arrow-left\"></i>Newer</a>", generate_slug(prev_entry))
		linkRowStr += prev_str
	}
	if idx < len(entries)-1 {
		next_entry := entries[idx+1]
		next_str := fmt.Sprintf("<a class=\"older-link\" href=\"%s\">Older<i class=\"fa fa-arrow-right\"/></i></a>", generate_slug(next_entry))
		linkRowStr += next_str
	}
	return template.HTML(linkRowStr)
}

func generate_unfurl(entry BlogEntry) template.HTML {
	unfurl := fmt.Sprintf(`<meta property="og:type"  content="article" />
		<meta property="og:url" content="https://gravitymoth.com/blog/%s" />
		<meta property="og:description" content="%s" />
		<meta property="og:title" content="%s" />
		<meta property="twitter:title" content="%s" />
		<meta property="og:image" content="https://gravitymoth.com/media/%s" />
		<meta property="og:locale" content="en_US">
		<meta property="og:site_name" content="Gravity Moth">
		<meta property="twitter:card" content="summary">`,
		entry.Slug,
		entry.Description,
		entry.Title,
		entry.Title,
		entry.Thumbnail,
	)

	return template.HTML(unfurl)
}

func generate_posts(entries []BlogEntry, html_templpath string) {
	tmpl, err := template.ParseFiles(html_templpath)
	if err != nil {
		panic(err)
	}

	slugs := make([]string, 0)
	for _, md := range entries {
		slug_link := generate_slug(md)
		li_str := fmt.Sprintf(`<a class="slug-entry" href="%s"><li><p>%s</p></li></a>`, slug_link, md.Title)
		slugs = append(slugs, li_str)
	}

	for i, entry := range entries {
		bin_name := fmt.Sprintf("%s%s.html", bin_dir, entry.Slug)
		if i == 0 {
			slug_link := fmt.Sprintf("%s", entry.Slug)
			headname := fmt.Sprintf("%sindex.html", bin_dir)
			generate_redirect(headname, slug_link)
		}

		f, err := os.Create(bin_name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		date_str := entry.Date.Format(out_time_fmt)
		hdr_str := fmt.Sprintf("<h1>%s</h1><h5>%s</h5>", entry.Title, date_str)

		slugs_str := ""
		for j, s := range slugs {
			if j == i {
				slugs_str += fmt.Sprintf(`<a class="slug-entry selected"><li><p>%s</p></li></a>`, entry.Title)
			} else {
				slugs_str += s
			}
		}

		data := PostTemplateData{
			Description: entry.Description,
			Unfurl:      generate_unfurl(entry),
			Header:      template.HTML(hdr_str),
			NavHead:     generate_link_row(entries, i),
			Slug:        fmt.Sprintf("slug-%s", entry.Slug),
			Content:     template.HTML(markdown.ToHTML(entry.Content, nil, nil)),
			Slugs:       template.HTML(slugs_str),
			NavFoot:     generate_link_row(entries, i),
		}

		err = tmpl.Execute(f, data)
		if err != nil {
			panic(err)
		}
	}
}

func generate_rss(entries []BlogEntry, rss_templpath string) {
	tmpl, err := template.ParseFiles(rss_templpath)
	if err != nil {
		panic(err)
	}

	entriesStr := ""
	for i, entry := range entries {
		date_str := entry.Date.Format(rss_time_fmt)
		post_link := fmt.Sprintf("https://gravitymoth.com/blog/%s", entry.Slug)
		post_str := fmt.Sprintf(`<item><title>%s</title><link>%s</link><description>%s</description><pubDate>%s</pubDate><guid isPermaLink="true">%s</guid></item>`, entry.Title, post_link, entry.Description, date_str, post_link)

		if i != 0 {
			entriesStr += "\n\t"
		}

		entriesStr += post_str
	}

	bin_name := fmt.Sprintf("%sfeed.xml", bin_dir)
	f, err := os.Create(bin_name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = tmpl.Execute(f, template.HTML(entriesStr))
	if err != nil {
		panic(err)
	}

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

			md := BlogEntry{Name: file.Name(), Path: static_dir, Content: content, Title: string(title), Date: date, Slug: string(slug), Description: string(desc), Thumbnail: string(img)}
			mds = append(mds, md)
		} else if file.Name() == "template.html" {
			html_template_file = file
		} else if file.Name() == "rss_template.xml" {
			rss_template_file = file
		}
	}

	if html_template_file == nil {
		log.Fatal("Couldn't find html template!\n")
	}
	if rss_template_file == nil {
		log.Fatal("Couldn't find rss template!\n")
	}

	_ = os.RemoveAll(bin_dir)
	_ = os.Mkdir(bin_dir, os.ModePerm)

	html_templpath := fmt.Sprintf("%s%s", static_dir, html_template_file.Name())
	rss_templpath := fmt.Sprintf("%s%s", static_dir, rss_template_file.Name())

	sort.SliceStable(mds, func(i, j int) bool {
		return mds[i].Date.After(mds[j].Date)
	})

	generate_posts(mds, html_templpath)
	generate_rss(mds, rss_templpath)

	fmt.Printf("site generated\n")
}
