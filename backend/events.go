package main

import (
"bufio"
"encoding/json"
"net/http"
"os/exec"
"strconv"
"strings"
"time"
)

type Event struct {
Ts    string `json:"ts"`    // RFC3339
Level string `json:"level"` // info/warn/error
Type  string `json:"type"`  // wg-easy
Msg   string `json:"msg"`
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
window := 400
if v := r.URL.Query().Get("window"); v != "" {
if n, err := strconv.Atoi(v); err == nil && n >= 50 && n <= 5000 {
window = n
}
}

// Pull wg-easy container logs
cmd := exec.Command("docker", "logs", "--tail", strconv.Itoa(window), "wg-easy")
out, err := cmd.Output()
if err != nil {
// If logs fail, still return empty list (do not break UI)
writeJSON(w, []Event{})
return
}

evs := parseWGEasyLogs(string(out))
writeJSON(w, evs)
}

func writeJSON(w http.ResponseWriter, v any) {
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(v)
}

func parseWGEasyLogs(logs string) []Event {
evs := []Event{}
sc := bufio.NewScanner(strings.NewReader(logs))

for sc.Scan() {
line := strings.TrimSpace(sc.Text())
if line == "" {
continue
}

// wg-easy typical: 2026-01-07T19:05:10.180Z WireGuard Config synced.
ts, msg := splitTS(line)

lower := strings.ToLower(msg)

// classify
switch {
case strings.Contains(lower, "server listening"):
evs = append(evs, Event{Ts: ts, Level: "info", Type: "wg-easy", Msg: "wg-easy web UI listening"})
case strings.Contains(lower, "wireguard loading configuration"):
evs = append(evs, Event{Ts: ts, Level: "info", Type: "wg-easy", Msg: "WireGuard loading configuration"})
case strings.Contains(lower, "config synced"):
evs = append(evs, Event{Ts: ts, Level: "info", Type: "wg-easy", Msg: "WireGuard config synced"})
case strings.Contains(lower, "wg-quick up"):
evs = append(evs, Event{Ts: ts, Level: "info", Type: "wg-easy", Msg: "wg-quick up wg0"})
case strings.Contains(lower, "wg-quick down"):
evs = append(evs, Event{Ts: ts, Level: "warn", Type: "wg-easy", Msg: "wg-quick down wg0"})
case strings.Contains(lower, "error") || strings.Contains(lower, "fail"):
evs = append(evs, Event{Ts: ts, Level: "error", Type: "wg-easy", Msg: msg})
}
}

// keep it small-ish
if len(evs) > 200 {
evs = evs[len(evs)-200:]
}
return evs
}

func splitTS(line string) (string, string) {
// if starts with RFC3339-ish timestamp
parts := strings.SplitN(line, " ", 2)
if len(parts) == 2 && looksLikeTS(parts[0]) {
return parts[0], parts[1]
}
// fallback: now
return time.Now().UTC().Format(time.RFC3339), line
}

func looksLikeTS(s string) bool {
// very loose check
if len(s) < 10 {
return false
}
// try parse
_, err := time.Parse(time.RFC3339Nano, s)
if err == nil {
return true
}
// wg-easy uses Z with millis, still RFC3339Nano parses, but keep fallback
return false
}

func getEvents(window string) ([]Event, error) {
w := 400
if window != "" {
if n, err := strconv.Atoi(window); err == nil && n >= 50 && n <= 5000 {
w = n
}
}

cmd := exec.Command("docker", "logs", "--tail", strconv.Itoa(w), "wg-easy")
out, err := cmd.Output()
if err != nil {
return []Event{}, nil
}

evs := parseWGEasyLogs(string(out))
return evs, nil
}
