package main

import (
"net/http"
"os/exec"
)

func restartHandler(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodPost {
w.WriteHeader(http.StatusMethodNotAllowed)
return
}

if err := exec.Command("docker", "restart", "wg-easy").Run(); err != nil {
http.Error(w, err.Error(), 500)
return
}

w.WriteHeader(http.StatusNoContent)
}
