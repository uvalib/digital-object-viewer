package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type viewerData struct {
	URI       string
	StartPage int
}

// viewHandler takes the initial viewer request and determines what type of resource it is and
// hands the rendering off to the appropriate handler - or returns 404
func viewHandler(c *gin.Context) {
	srcPID := c.Param("pid")
	if isIiifCandidate(srcPID) {
		unitID := c.Query("unit")
		iiifURL := fmt.Sprintf("%s/%s", config.iiifURL, srcPID)
		if unitID != "" {
			iiifURL = fmt.Sprintf("%s?unit=%s", iiifURL, unitID)
		}
		log.Printf("Render %s as image", srcPID)
		viewImage(c, iiifURL)
		return
	}

	// not an image; try Apollo for WSLS...
	log.Printf("%s is not image; check WSLS", srcPID)
	wslsData, err := getApolloWSLSMetadata(srcPID)
	if err == nil {
		log.Printf("Render %s as WSLS", srcPID)
		viewWSLS(c, wslsData)
		return
	}

	// Nope; fail
	c.String(http.StatusNotFound, "%s not found", srcPID)
}

// viewImage displays a series of images in the universalViewer
func viewImage(c *gin.Context, iiifURL string) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		page = 1
	}
	data := viewerData{URI: iiifURL, StartPage: page - 1}
	c.HTML(http.StatusOK, "image_view.html", data)
}

// viewWSLS renders a custom view of WSLS content that includes video clips, transcripts and a poster
func viewWSLS(c *gin.Context, wslsData *wslsMetadata) {
	if wslsData.HasVideo {
		// POSTER: http://fedora01.lib.virginia.edu/wsls/{wslsID}/{wslsID}-poster.jpg
		// VIDEO (webm): http://fedora01.lib.virginia.edu/wsls/{wslsID}/{wslsID}.webm
		wslsData.VideoURL = fmt.Sprintf("%s/%s/%s.webm", config.fedoraURL, wslsData.WSLSID, wslsData.WSLSID)
		wslsData.PosterURL = fmt.Sprintf("%s/%s/%s-poster.jpg", config.fedoraURL, wslsData.WSLSID, wslsData.WSLSID)
	}

	if wslsData.HasScript {
		// PDF: http://fedora01.lib.virginia.edu/wsls/{wslsID}/{wslsID}.pdf
		// Thumb: http://fedora01.lib.virginia.edu/wsls/{wslsID}/{wslsID}-script-thumbnail.jpg
		// Transcript: http://fedora01.lib.virginia.edu/wsls/0003_1/0003_1.txt
		wslsData.PDFURL = fmt.Sprintf("%s/%s/%s.pdf", config.fedoraURL, wslsData.WSLSID, wslsData.WSLSID)
		wslsData.PDFThumbURL = fmt.Sprintf("%s/%s/%s-script-thumbnail.jpg", config.fedoraURL, wslsData.WSLSID, wslsData.WSLSID)
		wslsData.TranscriptURL = fmt.Sprintf("%s/%s/%s.txt", config.fedoraURL, wslsData.WSLSID, wslsData.WSLSID)
	}

	c.HTML(http.StatusOK, "wsls_view.html", wslsData)
}

// Hit the target IIIF manifest URL and see if it contains any images
func isIiifCandidate(pid string) bool {
	log.Printf("Check if %s is a candidate for IIIF metadata...", pid)
	timeout := time.Duration(10 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(fmt.Sprintf("%s/%s/exist", config.iiifURL, pid))
	if err != nil {
		log.Printf("ERROR: IIIF exist returned an error: %s", err.Error())
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: IIIF exist returned non-success status: %d", resp.StatusCode)
		return false
	}
	log.Printf("PID %s has an IIIF manifest", pid)
	return true
}
