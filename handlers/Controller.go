package Handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"openify/ConfigurationManager"
	"openify/Response"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"openify/Authentication"
	"openify/FilesManager"

	"github.com/dhowden/tag"
)

type Metadata struct {
	Title       string `json:"title"`
	Album       string `json:"album"`
	Artist      string `json:"artist"`
	AlbumArtist string `json:"album-artist"`
	Composer    string `json:"composer"`
	Year        int    `json:"year"`
	Genre       string `json:"genre"`
	Comment     string `json:"comment"`
	Codec       string `json:"codec"`
	Filename    string `json:"filename"`
	Success     bool   `json:"success"`
}

type Version struct {
	ControllerVersion    int  `json:"controller-version"`
	MinimumClientVersion int  `json:"minimum-client-version"`
	Success              bool `json:"success"`
}

type AboutResponse struct {
	Os string `json:"os"`
	Arch string `json:"arch"`
	GoVersion string `json:"go-version"`
	ControllerVersion Version `json:"controller-version"`
	ServerVersion string `json:"server-version"`
	Success bool `json:"success"`
}

var version = Version{
	ControllerVersion:    1,
	MinimumClientVersion: 1,
	Success:              true,
}

func GetFilesList(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(FilesManager.GetRoot())
	if err != nil {
		log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
		authentication.SendError(w, r, err.Error())
		return
	}
	log.Printf("[INFO][%s] <--  List of files\n", r.RemoteAddr)
	Response.SendJson(w, r, b)
}

func GetFilePathFromID(r *http.Request) (string, error) {
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		return "", errors.New("file ID missing")
	}
	id, err := strconv.Atoi(ids[0])
	if err != nil {
		return "", errors.New("file ID is NaN")
	}
	path, err := FilesManager.GetPathById(id)
	if err != nil {
		return "", errors.New("file ID is not found")
	}
	return path, nil
}

func GetFile(w http.ResponseWriter, r *http.Request) {
	tokens, ok := r.URL.Query()["t"]
	if !ok || len(tokens[0]) < 1 {
		authentication.SendUnauthorized(w, r)
		return
	}
	token:= tokens[0]
	if authentication.IsLogged(token) {
		path, err := GetFilePathFromID(r)
		if err != nil {
			log.Printf("[ERROR][%s] %s\n", r.RemoteAddr, err)
			authentication.SendError(w, r, err.Error())
			return
		}
		log.Printf("[INFO][SERVING][%s] <-- %s\n", r.RemoteAddr, path)
		http.ServeFile(w, r, path)
	} else {
		authentication.SendUnauthorized(w, r)
	}
}

func GetMetaData(w http.ResponseWriter, r *http.Request) {
	path, err := GetFilePathFromID(r)
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
			Success:  true,
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
	log.Printf("[INFO][%s] <--  File metadata\n", r.RemoteAddr)
	Response.SendJson(w, r, b)
}

func ToOpenifyMetadata(md tag.Metadata, fn string) Metadata {
	return Metadata{
		Title:       md.Title(),
		Album:       md.Album(),
		Artist:      md.Artist(),
		AlbumArtist: md.AlbumArtist(),
		Composer:    md.Composer(),
		Year:        md.Year(),
		Genre:       md.Genre(),
		Comment:     md.Comment(),
		Codec:       string(md.FileType()),
		Filename:    fn,
		Success:     true,
	}
}

func GetControllerVersion(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(version)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		authentication.SendError(w, r, err.Error())
		return
	}
	log.Printf("[INFO][%s] <--  Controller version\n", r.RemoteAddr)
	Response.SendJson(w, r, b)
}

func ReScanFolder(w http.ResponseWriter, r *http.Request) {
	config := ConfigurationManager.GetConfiguration()
	FilesManager.ScanFolder(config.DocumentRoot)
	GetFilesList(w, r)
}

func About(w http.ResponseWriter, r *http.Request) {
	about:= AboutResponse{
		Os: runtime.GOOS,
		Arch: runtime.GOARCH,
		GoVersion: runtime.Version(),
		ControllerVersion: version,
		ServerVersion: ConfigurationManager.GetVersion(),
		Success: true,
	}

	b, err := json.Marshal(about)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		authentication.SendError(w, r, err.Error())
		return
	}
	log.Printf("[INFO][%s] <--  About\n", r.RemoteAddr)
	Response.SendJson(w, r, b)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header:= r.Header.Get("authorization")
		if len(header) > 7 {
			token:= header[7:]
			if authentication.IsLogged(token) {
				next.ServeHTTP(w, r)
			} else {
				authentication.SendUnauthorized(w, r)
			}
		} else {
			authentication.SendUnauthorized(w, r)
		}
	})
}

func HandleRequests() {
	config := ConfigurationManager.GetConfiguration()
	log.Printf("[INFO] Server listening at %s\n", config.Port)

	mux:= http.NewServeMux()
	mux.HandleFunc("/api/login", authentication.Login)
	mux.HandleFunc("/api/get/file", GetFile)
	mux.Handle("/api/list/files", AuthMiddleware(http.HandlerFunc(GetFilesList)))
	mux.Handle("/api/get/metadata", AuthMiddleware(http.HandlerFunc(GetMetaData)))
	mux.Handle("/api/system/server/about", AuthMiddleware(http.HandlerFunc(About)))
	mux.Handle("/api/system/files/scan", AuthMiddleware(http.HandlerFunc(ReScanFolder)))
	mux.Handle("/api/system/user/register", AuthMiddleware(http.HandlerFunc(authentication.Register)))
	mux.Handle("/api/system/controller/version", AuthMiddleware(http.HandlerFunc(GetControllerVersion)))
	mux.Handle("/api/system/user/list", AuthMiddleware(http.HandlerFunc(authentication.GetListUsers)))

	err := http.ListenAndServe(fmt.Sprintf(":%s", config.Port), mux)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
	}
}
