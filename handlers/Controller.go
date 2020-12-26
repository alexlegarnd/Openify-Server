package handlers

import (
	"Openify/authentication"
	"Openify/managers"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dhowden/tag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type Metadata struct {
	Title string `json:"title"`
	Album string `json:"album"`
	Artist string `json:"artist"`
	AlbumArtist string `json:"album-artist"`
	Composer string `json:"composer"`
	Year int `json:"year"`
	Genre string `json:"genre"`
	Comment string `json:"comment"`
	Codec string `json:"codec"`
	Filename string `json:"filename"`
	Success bool `json:"success"`
}

type Version struct {
	ControllerVersion int `json:"controller-version"`
	MinimumClientVersion int `json:"minimum-client-version"`
	Success bool `json:"success"`
}

var version = Version{
	ControllerVersion: 1,
	MinimumClientVersion: 1,
	Success: true,
}

func GetFilesList(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(managers.GetRoot())
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
		authentication.SendError(w, r, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	} else {
		log.Printf("[INFO][%s] Getting list of files\n", r.RemoteAddr)
	}
}

func GetFilePathFromID(r *http.Request) (string, error) {
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		return "", errors.New("file ID missing")
	}
	id, err:= strconv.Atoi(ids[0])
	if err != nil {
		return "", errors.New("file ID is NaN")
	}
	path, err:= managers.GetPathById(id)
	if err != nil {
		return "", errors.New("file ID is not found")
	}
	return path, nil
}

func GetFile(w http.ResponseWriter, r *http.Request) {
	path, err:= GetFilePathFromID(r)
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
		authentication.SendError(w, r, err.Error())
		return
	}
	log.Printf("[INFO][%s] Serving file %s\n", r.RemoteAddr, path)
	http.ServeFile(w, r, path)
}

func GetMetaData(w http.ResponseWriter, r *http.Request) {
	path, err:= GetFilePathFromID(r)
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
		authentication.SendError(w, r, err.Error())
		return
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
		return
	}
	var metaDataResult Metadata
	m, err := tag.ReadFrom(f)
	if err != nil {
		log.Printf("[WARN][%s] %s\n", r.RemoteAddr, err)
		metaDataResult = Metadata{
			Filename: filepath.Base(path),
			Success: true,
		}
	} else {
		metaDataResult = ToOpenifyMetadata(m, filepath.Base(path))
	}
	b, err := json.Marshal(metaDataResult)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		authentication.SendError(w, r, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	} else {
		log.Printf("[INFO][%s] Getting Metadata for %s\n", r.RemoteAddr, path)
	}
}

func ToOpenifyMetadata(md tag.Metadata, fn string) Metadata {
	return Metadata{
		Title: md.Title(),
		Album: md.Album(),
		Artist: md.Artist(),
		AlbumArtist: md.AlbumArtist(),
		Composer: md.Composer(),
		Year: md.Year(),
		Genre: md.Genre(),
		Comment: md.Comment(),
		Codec: string(md.FileType()),
		Filename: fn,
		Success: true,
	}
}

func GetControllerVersion(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(version)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		authentication.SendError(w, r, err.Error())
		return
	}
	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
	} else {
		log.Printf("[INFO][%s] Getting controller version\n", r.RemoteAddr)
	}
}

func ReScanFolder(w http.ResponseWriter, r *http.Request) {
	config:= managers.GetConfiguration()
	managers.ScanFolder(config.DocumentRoot)
	GetFilesList(w, r)
}

func HandleRequests() {
	config:= managers.GetConfiguration()
	log.Printf("[INFO] Server listening at %s\n", config.Port)
	http.HandleFunc("/api/login", authentication.Signin)
	http.HandleFunc("/api/list/files", GetFilesList)
	http.HandleFunc("/api/get/file", GetFile)
	http.HandleFunc("/api/get/metadata", GetMetaData)
	http.HandleFunc("/api/system/controller/version", GetControllerVersion)
	http.HandleFunc("/api/system/files/scan", ReScanFolder)
	err:= http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
	}
}
