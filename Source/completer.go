package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
)

type completionFileServer struct {
	fileSystem http.FileSystem
	fileServer http.Handler
}

func newCompletionFileServer(root http.FileSystem) http.Handler {
	return &completionFileServer{
		fileSystem: root,
		fileServer: http.FileServer(root),
	}
}

func (server *completionFileServer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	upath := server.cleanPath(request)
	info, err := server.getFileInfo(upath)
	if err == nil && server.checkRangeAtEnd(info, request) {
		response.WriteHeader(http.StatusOK)
	} else {
		server.fileServer.ServeHTTP(response, request)
	}
}

func (server *completionFileServer) cleanPath(request *http.Request) string {
	upath := request.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		request.URL.Path = upath
	}
	return path.Clean(upath)
}

func (server *completionFileServer) getFileInfo(upath string) (os.FileInfo, error) {
	file, err := server.fileSystem.Open(upath)
	if err != nil {
		return nil, err
	}
	return file.Stat()
}

func (server *completionFileServer) checkRangeAtEnd(info os.FileInfo, request *http.Request) bool {
	return request.Header.Get("Range") == fmt.Sprintf("bytes=%d-", info.Size())
}
