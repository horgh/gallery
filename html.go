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
	Name     string
}

const css = `
body {
	margin: 0;
	padding: 0;
}

#albums {
	text-align: center;
}

.album {
	display: inline-block;
	width: 250px;
	max-width: 250px;
	text-align: left;
}

.album img {
	display: inline-block;
}

.album p {
	display: inline-block;
	vertical-align: top;
	padding: 0;
	margin: 0;
	text-align: left;
	max-width: 140px;
}

#nav {
	margin: 15px 0 15px 0;
}

#images {
	margin: 0 50px 15px 50px;
}

.image {
	display: inline-block;
}

@media all and (max-width: 600px) {
  #images {
    margin: 0 0 15px 0;
  }
}
`

// makeGalleryHTML creates an HTML file that acts as the top level of the
// gallery. This is a single page that links to all albums.
func makeGalleryHTML(installDir, name string, albums []HTMLAlbum,
	verbose, forceGenerate bool) error {
	htmlPath := path.Join(installDir, "index.html")
	exists, err := fileExists(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to check if HTML exists: %s: %s", htmlPath, err)
	}

	if !forceGenerate && exists {
		return nil
	}

	err = makeDirIfNotExist(installDir)
	if err != nil {
		return err
	}

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
<title>{{.Name}}</title>
<meta name="viewport" content="width=device-width, user-scalable=no">
<style>` + css + `</style>
<h1>{{.Name}}</h1>

<div id="albums">
	{{range .Albums}}
		<div class="album">
			<a href="{{.URL}}"><img src="{{.ThumbURL}}"></a>
			<p><a href="{{.URL}}">{{.Name}}</a></p>
		</div>
	{{end}}
</div>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("unable to parse HTML template: %s", err)
	}

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("unable to open HTML file: %s", err)
	}

	data := struct {
		Name   string
		Albums []HTMLAlbum
	}{
		Name:   name,
		Albums: albums,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("close: %s", err)
	}

	if verbose {
		log.Printf("Wrote HTML file: %s", htmlPath)
	}
	return nil
}

// generate and write an HTML page for an album.
//
// This is the top level page of an album and shows potentially multiple images.
//
// galleryName is optional. It may be we are creating a standalone album.
func makeAlbumPageHTML(totalPages, totalImages, page int,
	images []HTMLImage, installDir, name, galleryName string,
	verbose, forceGenerate bool) error {
	// Figure out filename to write.
	// Page 1 is index.html. The rest are page-n.html
	filename := "index.html"
	if page > 1 {
		filename = fmt.Sprintf("page-%d.html", page)
	}

	htmlPath := path.Join(installDir, filename)
	exists, err := fileExists(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to check if HTML exists: %s: %s", htmlPath, err)
	}

	if !forceGenerate && exists {
		return nil
	}

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
{{if .GalleryName}}
<title>{{.Name}} - {{.GalleryName}}</title>
{{else}}
<title>{{.Name}}</title>
{{end}}
<meta name="viewport" content="width=device-width, user-scalable=no">
<style>` + css + `</style>
<h1>{{.Name}} ({{.TotalImages}} images)</h1>

<div id="nav">
	Navigation:
	{{if .GalleryName}}
		<a href="..">Back to {{.GalleryName}}</a> |
	{{end}}

	{{if gt .Page 1}}
		<a href="{{.PreviousURL}}">Previous page</a> |
	{{else}}
		Previous page |
	{{end}}

	{{if lt .Page .TotalPages}}
		<a href="{{.NextURL}}">Next page</a>
	{{else}}
		Next page
	{{end}}

	{{if gt .TotalPages 1}}
		(This is page {{.Page}}/{{.TotalPages}})
	{{end}}
</div>

<div id="images">
	{{range .Images}}
		<div class="image">
			<a href="image-{{.Index}}.html">
				<img src="{{.ThumbImageURL}}">
			</a>
		</div>
	{{end}}
</div>

<a href="{{.Name}}.zip">Download all images (.zip)</a>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("unable to parse HTML template: %s", err)
	}

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("unable to open HTML file: %s", err)
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
		Name        string
		GalleryName string
		Images      []HTMLImage
		TotalPages  int
		Page        int
		TotalImages int
		PreviousURL string
		NextURL     string
	}{
		Name:        name,
		GalleryName: galleryName,
		Images:      images,
		TotalPages:  totalPages,
		Page:        page,
		TotalImages: totalImages,
		PreviousURL: previousURL,
		NextURL:     nextURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("close: %s", err)
	}

	if verbose {
		log.Printf("Wrote HTML file: %s", htmlPath)
	}
	return nil
}

// Make an HTML page showing a single image.
//
// This page shows the larger size of the image. We link to the original image.
//
// galleryName is optional. It may be we are creating a standalone album.
func makeImagePageHTML(image HTMLImage, dir string, totalImages int,
	albumName, galleryName string, verbose, forceGenerate bool) error {
	htmlPath := path.Join(dir, fmt.Sprintf("image-%d.html", image.Index))
	exists, err := fileExists(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to check if HTML exists: %s: %s", htmlPath, err)
	}

	if !forceGenerate && exists {
		return nil
	}

	const tpl = `<!DOCTYPE html>
<meta charset="utf-8">
{{if .GalleryName}}
<title>{{.ImageName}} - {{.AlbumName}} - {{.GalleryName}}</title>
{{else}}
<title>{{.ImageName}} - {{.AlbumName}}</title>
{{end}}
<meta name="viewport" content="width=device-width, user-scalable=no">
<style>` + css + `</style>
<script>
"use strict";

var G = {};

document.addEventListener('DOMContentLoaded', function() {
	document.addEventListener('keydown', function(evt) {
		evt.preventDefault();

		{{if .PreviousURL}}
			// Left arrow key.
			if (evt.keyCode === 37) {
				window.location.href = "{{.PreviousURL}}";
				return;
			}
		{{end}}

		{{if .NextURL}}
			// Right arrow key.
			if (evt.keyCode === 39) {
				window.location.href = "{{.NextURL}}";
				return;
			}
		{{end}}
	});
});
</script>
<h1>{{.ImageName}}</h1>

<div id="nav">
	Navigation:
	{{if .GalleryName}}
	<a href="..">Back to {{.GalleryName}}</a> |
	{{end}}
	<a href="index.html">Back to {{.AlbumName}}</a>

	{{if .PreviousURL}}
		| <a href="{{.PreviousURL}}">Previous image</a>
	{{else}}
		| Previous image
	{{end}}

	{{if .NextURL}}
		| <a href="{{.NextURL}}">Next image</a>
	{{else}}
		| Next image
	{{end}}
</div>

<div class="image-large">
	<a href="{{.OriginalImageURL}}">
		<img src="{{.FullImageURL}}">
	</a>

	{{if .Description}}
		<p>{{.Description}}</p>
	{{end}}
</div>
`

	t, err := template.New("page").Parse(tpl)
	if err != nil {
		return fmt.Errorf("unable to parse HTML template: %s", err)
	}

	fh, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("unable to open HTML file: %s", err)
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
		AlbumName        string
		GalleryName      string
		OriginalImageURL string
		FullImageURL     string
		Description      string
		NextURL          string
		PreviousURL      string
	}{
		ImageName:        image.OriginalImageURL,
		AlbumName:        albumName,
		GalleryName:      galleryName,
		OriginalImageURL: image.OriginalImageURL,
		FullImageURL:     image.FullImageURL,
		Description:      image.Description,
		NextURL:          nextURL,
		PreviousURL:      previousURL,
	}

	err = t.Execute(fh, data)
	if err != nil {
		_ = fh.Close()
		return fmt.Errorf("unable to execute template: %s", err)
	}

	err = fh.Close()
	if err != nil {
		return fmt.Errorf("close: %s", err)
	}

	if verbose {
		log.Printf("Wrote HTML file: %s", htmlPath)
	}
	return nil
}
