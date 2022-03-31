package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type info struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Code    string `json:"code"`
	City    string `json:"city"`
}

type File struct {
	ModTime  string  `json:"mod-time"`
	IsDir    bool    `json:"is-dir"`
	Size     int64   `json:"size"`
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Children []*File `json:"children"`
}

type Status struct {
	Status string                 `json:"status"`
	Out    string                 `json:"stdout"`
	Result map[string]interface{} `json:"jsonst"`
}

func toFile(file os.FileInfo, path string) *File {
	JSONFile := File{
		ModTime: file.ModTime().Format("2006-01-02 15:04:05"),
		IsDir:   file.IsDir(),
		Size:    file.Size(),
		Name:    file.Name(),
		Path:    path,
	}
	return &JSONFile
}

func (a *App) getFilesTree(w http.ResponseWriter, r *http.Request) {

	rootOSFile, _ := os.Stat(os.Getenv("TREE_PATH"))
	rootFile := toFile(rootOSFile, os.Getenv("TREE_PATH"))
	stack := []*File{rootFile}

	for len(stack) > 0 {
		file := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		children, _ := ioutil.ReadDir(file.Path)
		for _, ch := range children {
			child := toFile(ch, filepath.Join(file.Path, ch.Name()))
			file.Children = append(file.Children, child)
			stack = append(stack, child)
		}
	}

	respondWithJSON(w, http.StatusOK, rootFile)
}

func (a *App) getFilesList(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	ep := vars["ep"]
	var list []*File

	files, err := ioutil.ReadDir(FilesPath + ep)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Not found")
	}

	for _, f := range files {
		if f.Size() > 1024*1024 {
			child := toFile(f, filepath.Join(FilesPath+ep, f.Name()))
			list = append(list, child)
		}
	}

	respondWithJSON(w, http.StatusOK, list)
}

func (a *App) putJson(w http.ResponseWriter, r *http.Request) {
	var s Status
	vars := mux.Vars(r)
	endpoint := vars["ep"]

	b, _ := ioutil.ReadAll(r.Body)

	cmd := exec.Command("/opt/exec/"+endpoint+".sh", string(b))
	cmd.Dir = "/opt/exec/"
	out, err := cmd.CombinedOutput()

	if err != nil {
		s.Out = err.Error()
	}

	s.Out = string(out)
	json.Unmarshal(out, &s.Result)

	defer r.Body.Close()

	if err != nil {
		s.Status = "error"
	} else {
		s.Status = "ok"
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) getClientInfo(w http.ResponseWriter, r *http.Request) {

	var i info

	i.IP = getRealIP(r)
	record, err := a.Geoip.City(net.ParseIP(i.IP))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "GeoIP not found")
		return
	}

	i.Country = record.Country.Names["en"]
	i.Code = record.Country.IsoCode
	i.City = record.City.Names["en"]

	respondWithJSON(w, http.StatusOK, i)
}

func getRealIP(r *http.Request) string {

	remoteIP := ""
	// the default is the originating ip. but we try to find better options because this is almost
	// never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteIP = parts[0]
	}
	// If we have a forwarded-for header, take the address from there
	if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		lastFwd := addrs[len(addrs)-1]
		if ip := net.ParseIP(lastFwd); ip != nil {
			remoteIP = ip.String()
		}
		// parse X-Real-Ip header
	} else if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			remoteIP = ip.String()
		}
	}

	return remoteIP
}
