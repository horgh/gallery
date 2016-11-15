package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

// HTMLImage holds image info needed in HTML.
type HTMLImage struct {
	FullImageURL  string
	ThumbImageURL string
	Description   string
}

// generateHTML does just that!
//
// Split over several pages if necessary.
func generateHTML(images []Image, resizedImageDir string, thumbSize int,
	fullSize int, installDir string, title string) error {
	var htmlImages []HTMLImage

	page := 1

	totalPages := len(images) / pageSize
	if len(images)%pageSize > 0 {
		totalPages++
	}

	for _, img := range images {
		thumbFilename, err := img.getResizedFilename(thumbSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine thumbnail filename: %s", err)
		}

		fullFilename, err := img.getResizedFilename(fullSize, resizedImageDir)
		if err != nil {
			return fmt.Errorf("Unable to determine full image filename: %s", err)
		}

		htmlImages = append(htmlImages, HTMLImage{
			FullImageURL:  filepath.Base(fullFilename),
			ThumbImageURL: filepath.Base(thumbFilename),
			Description:   img.Description,
		})

		if len(htmlImages) == pageSize {
			err = writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
				title)
			if err != nil {
				return fmt.Errorf("Unable to generate/write HTML: %s", err)
			}

			htmlImages = nil
			page++
		}
	}

	if len(htmlImages) > 0 {
		err := writeHTMLPage(totalPages, len(images), page, htmlImages, installDir,
			title)
		if err != nil {
			return fmt.Errorf("Unable to generate/write HTML: %s", err)
		}
	}

	return nil
}

// writeHTMLPage generates and writes an HTML page for the given set of images.
func writeHTMLPage(totalPages int, totalImages int, page int,
	images []HTMLImage, installDir string, title string) error {
	const tpl = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>{{.Title}}</title>
</head>
<body>
<h1>{{.Title}}</h1>
{{range .Images}}
<div class="image">
	<a href="{{.FullImageURL}}">
		<img src="{{.ThumbImageURL}}">
	</a>
	<p>{{.Description}}</p>
</div>
{{end}}
{{if gt .TotalPages 1}}
<p>This is page {{.Page}} of {{.TotalPages}} of images.</p>

{{if gt .Page 1}}
<p><a href="{{.PreviousURL}}">Previous page</a></p>
{{end}}

{{if lt .Page .TotalPages}}
<p><a href="{{.NextURL}}">Next page</a></p>
{{end}}

{{end}}
</body>
</html>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("Unable to parse HTML template: %s", err)
	}

	// Figure out filename to write.
	// Page 1 is index.html. The rest are page-n.html
	filename := "index.html"
	if page > 1 {
		filename = fmt.Sprintf("page-%d.html", page)
	}

	path := fmt.Sprintf("%s%c%s", installDir, os.PathSeparator, filename)

	fh, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Unable to open HTML file: %s", err)
	}
	defer fh.Close()

	previousURL := ""
	if page > 1 {
		if page == 2 {
			previousURL = "index.html"
		} else {
			previousURL = fmt.Sprintf("page-%d.html", page-1)
		}
	}

	nextURL := ""
	if page < totalPages {
		nextURL = fmt.Sprintf("page-%d.html", page+1)
	}

	data := struct {
		Title       string
		Images      []HTMLImage
		TotalPages  int
		Page        int
		PreviousURL string
		NextURL     string
	}{
		Title:       title,
		Images:      images,
		TotalPages:  totalPages,
		Page:        page,
		PreviousURL: previousURL,
		NextURL:     nextURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		return fmt.Errorf("Unable to execute template: %s", err)
	}

	log.Printf("Wrote HTML file: %s", filename)
	return nil
}
