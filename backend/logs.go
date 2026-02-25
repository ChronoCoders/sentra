package main

import (
"net/http"
"os/exec"
)

func logsHandler(w http.ResponseWriter, r *http.Request) {
tail := r.URL.Query().Get("tail")
if tail == "" {
tail = "100"
}

out, err := exec.Command("docker", "logs", "--tail", tail, "wg-easy").Output()
if err != nil {
http.Error(w, err.Error(), 500)
return
}

w.Header().Set("Content-Type", "text/plain")
w.Write(out)
}
