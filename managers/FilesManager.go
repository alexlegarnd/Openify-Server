package managers

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var supportedExt = [...]string { ".mp3", ".wav", ".ogg", ".oga", ".flac" }

type File struct {
	Name string `json:"name"`
	Ext  string `json:"ext"`
	Id int `json:"id"`
}

type Folder struct {
	Name    string   `json:"name"`
	Folders []Folder `json:"folders"`
	Files   []File   `json:"files"`
	Success bool `json:"success"`
}

type Reference struct {
	Id int
	Path string
}

var root = Folder {
	Name:    "/",
	Folders: []Folder{},
	Files:   []File{},
	Success: true,
}

var references []Reference

func ScanFolder(_rootPath string) Folder {
	total:= 0
	ClearRoot()
	if _, err := os.Stat(_rootPath); err != nil {
		log.Fatalf("[ERROR] DocumentRoot folder does not exist ::> %s\n%s", _rootPath, err)
	}
	log.Printf("[INFO] Scanning music folder (%s)", _rootPath)
	err := filepath.Walk(_rootPath, func(path string, info os.FileInfo, err error) error {
		for _, ext := range supportedExt {
			if strings.HasSuffix(path, ext) {
				AddToFolder(path[len(_rootPath)+1:], ext, total)
				total++
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
	}
	log.Printf("[INFO] %d files found", total)
	return root
}

func ClearRoot() {
	root.Folders = root.Folders[:0]
	root.Files = root.Files[:0]
}

func AddToFolder(path string, ext string, id int) {
	pathUnix := filepath.ToSlash(path)
	s:= strings.Split(pathUnix, "/")
	var f = &root
	if len(s) == 1 {
		Append(&f.Files, id, pathUnix, ext, path)
	} else {
		for _, fld := range s {
			if strings.HasSuffix(fld, ext) {
				Append(&f.Files, id, fld, ext, path)
			} else {
				predict:= Filter(f.Folders, fld)
				if len(predict) == 1 {
					f = predict[0]
				} else {
					nf:= Folder{
						Name:    fld,
						Folders: []Folder{},
						Files:   []File{},
					}
					f.Folders = append(f.Folders, nf)
					predict:= Filter(f.Folders, fld)
					if len(predict) == 1 {
						f = predict[0]
					}
				}
			}
		}
	}
}

func Append(files *[]File, id int, name string, ext string, path string) {
	*files = append(*files, File{
		Name: name,
		Ext:  ext,
		Id: id,
	})
	references = append(references, Reference{
		Id: id,
		Path: path,
	})
}

func Filter(folders []Folder, name string) []*Folder {
	var result []*Folder
	for i := range folders {
		if folders[i].Name == name {
			result = append(result, &folders[i])
		}
	}
	return result
}

func GetPathById(id int) (string, error) {
	result, err:= GetFilenameById(id)
	if err != nil {
		return "", errors.New("ID not found")
	}
	return GetAbsolutePath(result), nil
}

func GetFilenameById(id int) (string, error) {
	var result = ""
	for _, ref := range references {
		if ref.Id == id {
			result = ref.Path
		}
	}
	if result == "" {
		return "", errors.New("ID not found")
	}
	return result, nil
}

func GetAbsolutePath(path string) string {
	abs:= config.DocumentRoot
	if !strings.HasSuffix(abs, string(os.PathSeparator)) {
		abs = abs + string(os.PathSeparator)
	}
	return fmt.Sprintf("%s%s", abs, path)
}

func GetRoot() Folder {
	return root
}