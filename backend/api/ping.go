package api

import (
    "net/http"
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Bienvenido al ChatSphere"))
}