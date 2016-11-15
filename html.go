package gallery

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
)

// HTMLImage holds image info needed in HTML.
type HTMLImage struct {
	OriginalImageURL string
	FullImageURL     string
	ThumbImageURL    string
	Description      string
	Index            int
}

// HTMLAlbum holds info needed in HTML about an album.
type HTMLAlbum struct {
	URL      string
	ThumbURL string
	Title    string
}

// generate and write an HTML page for an album.
func makeAlbumPageHTML(totalPages, totalImages, page int,
	images []HTMLImage, installDir, title, galleryTitle string) error {

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
<title>{{.Title}} - {{.GalleryTitle}}</title>
<h1>{{.Title}}</h1>

<a href="..">{{.GalleryTitle}}</a>

{{range .Images}}
	<div class="image">
		<a href="image-{{.Index}}.html">
			<img src="{{.ThumbImageURL}}">
		</a>
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

	htmlPath := path.Join(installDir, filename)

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("Unable to open HTML file: %s", err)
	}

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
		Title        string
		GalleryTitle string
		Images       []HTMLImage
		TotalPages   int
		Page         int
		PreviousURL  string
		NextURL      string
	}{
		Title:        title,
		GalleryTitle: galleryTitle,
		Images:       images,
		TotalPages:   totalPages,
		Page:         page,
		PreviousURL:  previousURL,
		NextURL:      nextURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("Unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
	}

	log.Printf("Wrote HTML file: %s", htmlPath)
	return nil
}

// Make an HTML page showing a single image. We show the larger size of the
// image. We link to the original image.
func makeImagePageHTML(image HTMLImage, dir, albumName string,
	totalImages int, albumTitle, galleryTitle string) error {

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
<title>{{.ImageName}} - {{.AlbumTitle}} - {{.GalleryTitle}}</title>
<h1>{{.ImageName}}</h1>

<a href="..">{{.GalleryTitle}}</a> |
<a href="index.html">{{.AlbumTitle}}</a>

<div class="image-large">
	<a href="{{.OriginalImageURL}}">
		<img src="{{.FullImageURL}}">
	</a>

	{{if .Description}}
		<p>{{.Description}}</p>
	{{end}}
</div>

{{if .PreviousURL}}
	<a href="{{.PreviousURL}}">Previous</a>
{{end}}

{{if .NextURL}}
	<a href="{{.NextURL}}">Next</a>
{{end}}
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("Unable to parse HTML template: %s", err)
	}

	htmlPath := path.Join(dir, fmt.Sprintf("image-%d.html", image.Index))

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("Unable to open HTML file: %s", err)
	}

	nextURL := ""
	if image.Index < totalImages-1 {
		nextURL = fmt.Sprintf("image-%d.html", image.Index+1)
	}

	previousURL := ""
	if image.Index > 0 {
		previousURL = fmt.Sprintf("image-%d.html", image.Index-1)
	}

	data := struct {
		ImageName        string
		AlbumTitle       string
		GalleryTitle     string
		OriginalImageURL string
		FullImageURL     string
		Description      string
		NextURL          string
		PreviousURL      string
	}{
		ImageName:        image.OriginalImageURL,
		AlbumTitle:       albumTitle,
		GalleryTitle:     galleryTitle,
		OriginalImageURL: image.OriginalImageURL,
		FullImageURL:     image.FullImageURL,
		Description:      image.Description,
		NextURL:          nextURL,
		PreviousURL:      previousURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("Unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
	}

	log.Printf("Wrote HTML file: %s", htmlPath)
	return nil
}

// makeGalleryHTML creates an HTML file that acts as the top level of the
// gallery. This is a single page that links to all albums.
func makeGalleryHTML(installDir, name string, albums []HTMLAlbum) error {
	err := makeDirIfNotExist(installDir)
	if err != nil {
		return err
	}

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
<title>{{.Title}}</title>
<h1>{{.Title}}</h1>

{{range .Albums}}
	<div class="album">
		<a href="{{.URL}}">
			<img src="{{.ThumbURL}}">
		</a>
		<p>{{.Title}}</p>
	</div>
{{end}}
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("Unable to parse HTML template: %s", err)
	}

	htmlPath := path.Join(installDir, "index.html")

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("Unable to open HTML file: %s", err)
	}

	data := struct {
		Title  string
		Albums []HTMLAlbum
	}{
		Title:  name,
		Albums: albums,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("Unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("Close: %s", err)
	}

	log.Printf("Wrote HTML file: %s", htmlPath)
	return nil
}
