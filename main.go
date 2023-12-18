package main

import (
	"compress/gzip"

	json2 "encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func assertNotError(e error) {
	if e != nil {
		log.Printf("Error! %f\n", e)
	}
}

type Dms struct {
	Title     string        `json:"Title"`
	Resources []DmsResource `json:"Resources"`
}

type DmsResource struct {
	MimeType string `json:"MimeType"`
	Command  string `json:"Command"`
}

type Nfo struct {
	XMLName   xml.Name `xml:"movie"`
	Title     string   `xml:"title"`
	SortTitle string   `xml:"sorttitle"`
	Plot      string   `xml:"plot"`
	Thumb     string   `xml:"thumb"`
	Tag       []string `xml:"tag"`
	Premiered string   `xml:"premiered"`
	Directory string   `xml:"director"`
	Studio    string   `xml:"studio"`
	Genre     []string `xml:"genre"`
	Set       Set      `xml:"set"`
}

type Set struct {
	Name     string `xml:"name"`
	Overview string `xml:"overview"`
}

func updateArchive(id string) {
	archiveFile := flag.String("archive", "archive.txt", "Archive file")
	flag.Parse()
	if len(*archiveFile) > 0 {
		f, err := os.OpenFile(*archiveFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		assertNotError(err)
		defer f.Close()

		_, err = f.WriteString("youtube " + id + "\n")
		assertNotError(err)
	}
}

func imageFileExists(directory, basename string) bool {
	imageExtensions := []string{".webp", ".jpg", ".png"}
	for _, ext := range imageExtensions {

		filename := directory + "/" + basename + ext
		if _, err := os.Stat(filename); err == nil {
			return true
		}

	}
	return false
}

func downloadFile(URL, directory, basename string) (string, error) {
	response, err := http.Get(URL)
	if err != nil {
		if response != nil {
			fmt.Printf("Error while retrieving %s, status code is %d, error code is %v \n", URL, response.StatusCode, err)
		} else {
			fmt.Printf("Error while retrieving %s, error code is %v \n", URL, err)
		}
		return "", errors.New("Could not get " + URL)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New("Received non 200 response code")
	}

	contentType := response.Header.Get("Content-Type")
	var filename string
	switch contentType {
	case "image/webp":
		filename = directory + "/" + basename + ".webp"
	case "image/jpeg":
		filename = directory + "/" + basename + ".jpg"
	case "image/jpg":
		filename = directory + "/" + basename + ".jpg"
	case "image/png":
		filename = directory + "/" + basename + ".png"
	}

	if _, err := os.Stat(filename); err == nil {
		//idempotent
		return filepath.Base(filename), nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", err
	}
	return filepath.Base(filename), nil
}

func main() {

	var fetchThumb bool
	var forceFetchThumb bool
	var suppressThumb bool
	var forceNfo bool
	var forceStrm bool
	var forceDms bool
	var sleep int
	flag.BoolVar(&fetchThumb, "fetchthumb", false, "Retrieve the thumbnail image and refer to it as file rather than URL")
	flag.BoolVar(&forceFetchThumb, "forcefetchthumb", false, "Force fetch thumbnail image even if already fetched")
	flag.BoolVar(&suppressThumb, "suppressthumb", false, "Suppress thumbnail image")
	flag.BoolVar(&forceNfo, "forcenfo", false, "Force creation of nfo")
	flag.BoolVar(&forceStrm, "forcestrm", false, "Force creation of strm")
	flag.BoolVar(&forceDms, "forceDms", false, "Force creation dms.json")
	flag.IntVar(&sleep, "sleep", 0, "Sleep before parsing")
	flag.Parse()

	if len(flag.Args()) <= 1 {
		fmt.Errorf("missing arguments")
	}
	filename := flag.Args()[0]

	fmt.Printf("Converting %s, fetchthumb=%v, suppressthumb=%v, forcenfo=%v, forcestrm=%v, forcedms=%v\n", filename, fetchThumb, suppressThumb, forceNfo, forceStrm, forceDms)

	filenameRegex := regexp.MustCompile(`(.*.info.json)(\.gz)?`)
	filenameMatches := filenameRegex.FindStringSubmatch(filepath.Base(filename))
	if len(filenameMatches) > 0 {
		directory := filepath.Dir(filename)
		fullBase := filenameMatches[0]
		base := filenameMatches[1]
		isGzip := strings.HasSuffix(fullBase, ".gz")

		//fmt.Printf("base %s \n", base)
		//fmt.Printf("r %v fullbase %v is gzip %v\n", filenameMatches, fullBase, isGzip)

		strmFilePath := directory + "/" + base + ".strm"
		nfoFilePath := directory + "/" + base + ".nfo"
		dmsFilePath := directory + "/" + base + ".dms.json"
		createNfo := false
		createStrm := false
		createDms := false
		if _, err := os.Stat(strmFilePath); err == nil {
			fmt.Printf("STRM %s exist, skipping\n", strmFilePath)
			//Don't need to create
		} else if os.IsNotExist(err) {
			createStrm = true
		}
		if _, err := os.Stat(nfoFilePath); err == nil {
			fmt.Printf("NFO %s exist, skipping\n", nfoFilePath)
			//Don't need to create
		} else if os.IsNotExist(err) {
			createNfo = true
		}
		if _, err := os.Stat(dmsFilePath); err == nil {
			fmt.Printf("DMS %s exist, skipping\n", dmsFilePath)
			//Don't need to create
		} else if os.IsNotExist(err) {
			createDms = true
		}

		if createStrm || createNfo || forceNfo || forceStrm || forceFetchThumb || createDms {

			var json []byte
			var err error
			if isGzip {
				json, err = readGzFile(filename)
			} else {
				json, err = ioutil.ReadFile(filename)
			}
			assertNotError(err)

			extractorKey := gjson.GetBytes(json, "extractor_key")
			id := gjson.GetBytes(json, "id")
			uploadDate := gjson.GetBytes(json, "upload_date")
			uploader := gjson.GetBytes(json, "uploader")
			playlistUploader := gjson.GetBytes(json, "playlist_uploader")
			// series := gjson.GetBytes(json, "series") SVT
			// playlist := gjson.GetBytes(json, "playlist") SVT
			playlistTitle := gjson.GetBytes(json, "playlist_title")
			title := gjson.GetBytes(json, "title")
			// fullTitle := gjson.GetBytes(json, "fulltitle") SVT

			description := gjson.GetBytes(json, "description")
			thumbnailStr := ""

			if !suppressThumb && (createNfo || forceNfo) {
				thumbnail := gjson.GetBytes(json, "thumbnail")
				if !thumbnail.Exists() {
					thumbnail = gjson.GetBytes(json, "thumbnails.#(width>=640)#.first.url")
				}
				if !thumbnail.Exists() {
					thumbnail = gjson.GetBytes(json, "thumbnails.3.url")
				}
				if !thumbnail.Exists() {
					thumbnail = gjson.GetBytes(json, "thumbnails.2.url")
				}
				if !thumbnail.Exists() {
					thumbnail = gjson.GetBytes(json, "thumbnails.1.url")
				}
				if !thumbnail.Exists() {
					thumbnail = gjson.GetBytes(json, "thumbnails.0.url")
				}
				if fetchThumb {

					if forceFetchThumb || !imageFileExists(directory, base) {
						fmt.Printf("Fetching thumbnail for %s\n", base)
						time.Sleep(time.Duration(sleep) * time.Second) //Sleep to slow down when fetching the file
						thumbnailStr, err = downloadFile(thumbnail.Str, directory, base)
						if err != nil {
							fmt.Printf("Error while fetching thumbnail for %s, %v", base, err)
							thumbnailStr = ""
						}
					} else {
						fmt.Printf("Image file for %s already exist, will not fetch\n", base)
					}
				} else {
					fmt.Printf("Not fetching thumbnail")
					thumbnailStr = thumbnail.Str
				}
			}
			channel := gjson.GetBytes(json, "channel")
			categorylist := []string{}
			categories := gjson.GetBytes(json, "categories")
			for _, category := range categories.Array() {
				categorylist = append(categorylist, category.Str)
			}
			taglist := []string{}
			taglist = append(taglist, uploader.Str)
			tags := gjson.GetBytes(json, "tags")
			for _, tag := range tags.Array() {
				taglist = append(taglist, tag.Str)
			}

			tm, _ := time.Parse("20060102", uploadDate.Str)

			if createStrm || forceStrm {
				var strm []byte
				switch extractorKey.Str {
				case "Youtube":
					strm = []byte(fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s\n", id))
				case "SVTPlay":
					url := gjson.GetBytes(json, "formats.1.manifest_url")
					if url.Exists() {
						strm = []byte(url.Str + "\n")
					}
				}
				if len(strm) > 0 {
					err = ioutil.WriteFile(strmFilePath, strm, 0644)
					assertNotError(err)
					err = os.Chtimes(strmFilePath, tm, tm)
					assertNotError(err)
				}
			}

			if createNfo || forceNfo || forceFetchThumb || createDms {
				var nfo *Nfo
				var dms *Dms
				switch extractorKey.Str {
				case "Youtube":
					set := Set{Name: fmt.Sprintf("%s : %s ", channel, playlistTitle)}
					nfo = &Nfo{Title: title.Str, SortTitle: title.Str, Plot: description.Str, Thumb: thumbnailStr, Genre: categorylist,
						Premiered: tm.Format("2006-01-02"), Tag: taglist, Directory: uploader.Str, Studio: playlistUploader.Str, Set: set}

					// command := fmt.Sprintf("yt-dlp --proxy http://127.0.0.1:6666 -f 22 %s -o -", id)
					command := fmt.Sprintf("play-stream %s", id)
					dms = &Dms{Title: title.Str, Resources: []DmsResource{{MimeType: "video/mp4", Command: command}}}
				case "SVTPlay":
					// "series": "Lokala Nyheter Örebro
					// "playlist": "Lokala Nyheter Örebro",
					// "playlist_title": "Lokala Nyheter Örebro",
					// "fulltitle": "6 aug. 07.35",
					set := Set{Name: fmt.Sprintf("%s : %s ", "SVT", playlistTitle)}
					svtTitle := fmt.Sprintf("%s %s", playlistTitle, title.Str)
					nfo = &Nfo{Title: svtTitle, SortTitle: svtTitle, Premiered: tm.Format("2006-01-02"), Set: set}
				}

				if nfo != nil {
					nfoFile, _ := os.Create(nfoFilePath)
					assertNotError(err)

					xmlWriter := io.Writer(nfoFile)
					enc := xml.NewEncoder(xmlWriter)
					err := enc.Encode(nfo)
					assertNotError(err)

					err = os.Chtimes(nfoFilePath, tm, tm)
					assertNotError(err)
				}

				if createDms && dms != nil {
					dmsFile, _ := os.Create(dmsFilePath)
					defer dmsFile.Close()
					assertNotError(err)

					jsonData, err := json2.Marshal(dms)
					assertNotError(err)

					_, err = dmsFile.Write(jsonData)
					assertNotError(err)

					err = os.Chtimes(dmsFilePath, tm, tm)
					assertNotError(err)
				}

			}

			//			if createStrm || createNfo {
			//				updateArchive(id.Str)
			//			}
		}
	}
}

func readGzFile(filename string) ([]byte, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return nil, err
	}
	defer fz.Close()

	s, err := ioutil.ReadAll(fz)
	if err != nil {
		return nil, err
	}
	return s, nil
}
