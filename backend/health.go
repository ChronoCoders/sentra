package main

import (
"bufio"
"encoding/json"
"net/http"
"os"
"os/exec"
"strconv"
"strings"
"sync"
"syscall"
"time"
)

type HostHealth struct {
UptimeSec   int64   `json:"uptime_sec"`
CPUPercent  float64 `json:"cpu_percent"`
Load1       float64 `json:"load1"`
Load5       float64 `json:"load5"`
Load15      float64 `json:"load15"`
MemTotalMB  int64   `json:"mem_total_mb"`
MemUsedMB   int64   `json:"mem_used_mb"`
DiskTotalGB float64 `json:"disk_total_gb"`
DiskUsedGB  float64 `json:"disk_used_gb"`
}

type DockerHealth struct {
Name         string `json:"name"`
Status       string `json:"status"`        // running/exited
Health       string `json:"health"`        // healthy/unhealthy/starting/none
RestartCount int64  `json:"restart_count"` // docker restarts
Image        string `json:"image"`
CreatedAt    string `json:"created_at"`
}

type HealthResponse struct {
Host   HostHealth   `json:"host"`
WGEasy DockerHealth `json:"wg_easy"`
}

var cpuMu sync.Mutex
var prevCPU = cpuStat{}
var prevCPUAt time.Time

type cpuStat struct {
idle  uint64
total uint64
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
host := readHostHealth()
wg := readDockerHealth("wg-easy")

resp := HealthResponse{Host: host, WGEasy: wg}

w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(resp)
}

func readHostHealth() HostHealth {
h := HostHealth{}

// uptime
if b, err := os.ReadFile("/proc/uptime"); err == nil {
fields := strings.Fields(string(b))
if len(fields) >= 1 {
if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
h.UptimeSec = int64(v)
}
}
}

// loadavg
if b, err := os.ReadFile("/proc/loadavg"); err == nil {
fields := strings.Fields(string(b))
if len(fields) >= 3 {
h.Load1, _ = strconv.ParseFloat(fields[0], 64)
h.Load5, _ = strconv.ParseFloat(fields[1], 64)
h.Load15, _ = strconv.ParseFloat(fields[2], 64)
}
}

// meminfo
memTotalKB, memAvailKB := int64(0), int64(0)
f, err := os.Open("/proc/meminfo")
if err == nil {
defer f.Close()
sc := bufio.NewScanner(f)
for sc.Scan() {
line := sc.Text()
if strings.HasPrefix(line, "MemTotal:") {
memTotalKB = parseMeminfoKB(line)
}
if strings.HasPrefix(line, "MemAvailable:") {
memAvailKB = parseMeminfoKB(line)
}
}
}
if memTotalKB > 0 {
h.MemTotalMB = memTotalKB / 1024
usedKB := memTotalKB - memAvailKB
if usedKB < 0 {
usedKB = 0
}
h.MemUsedMB = usedKB / 1024
}

// disk /
var st syscall.Statfs_t
if err := syscall.Statfs("/", &st); err == nil {
total := float64(st.Blocks) * float64(st.Bsize)
free := float64(st.Bavail) * float64(st.Bsize)
used := total - free
h.DiskTotalGB = total / (1024 * 1024 * 1024)
h.DiskUsedGB = used / (1024 * 1024 * 1024)
}

// cpu percent (delta from /proc/stat)
h.CPUPercent = readCPUPercent()

return h
}

func parseMeminfoKB(line string) int64 {
// format: "MemTotal:       16367460 kB"
fields := strings.Fields(line)
if len(fields) >= 2 {
v, _ := strconv.ParseInt(fields[1], 10, 64)
return v
}
return 0
}

func readCPUPercent() float64 {
// Read /proc/stat first line: cpu  user nice system idle iowait irq softirq steal guest guest_nice
b, err := os.ReadFile("/proc/stat")
if err != nil {
return 0
}
line := ""
for _, l := range strings.Split(string(b), "\n") {
if strings.HasPrefix(l, "cpu ") {
line = l
break
}
}
if line == "" {
return 0
}
fields := strings.Fields(line)
if len(fields) < 5 {
return 0
}

var vals []uint64
for _, f := range fields[1:] {
v, err := strconv.ParseUint(f, 10, 64)
if err != nil {
v = 0
}
vals = append(vals, v)
}

idle := vals[3]
if len(vals) >= 5 {
idle += vals[4] // iowait
}
var total uint64
for _, v := range vals {
total += v
}

cur := cpuStat{idle: idle, total: total}

cpuMu.Lock()
defer cpuMu.Unlock()

// first call
if prevCPUAt.IsZero() {
prevCPU = cur
prevCPUAt = time.Now()
return 0
}

prev := prevCPU
prevCPU = cur
prevCPUAt = time.Now()

dTotal := float64(cur.total - prev.total)
dIdle := float64(cur.idle - prev.idle)
if dTotal <= 0 {
return 0
}
usage := (dTotal - dIdle) / dTotal * 100
if usage < 0 {
usage = 0
}
if usage > 100 {
usage = 100
}
return usage
}

func readDockerHealth(container string) DockerHealth {
d := DockerHealth{Name: container}

// Inspect JSON (single shot, robust enough)
out, err := exec.Command("docker", "inspect", container).Output()
if err != nil {
d.Status = "not_found"
d.Health = "none"
return d
}

// Minimal parsing with map to avoid heavy struct
var arr []map[string]any
if err := json.Unmarshal(out, &arr); err != nil || len(arr) == 0 {
d.Status = "unknown"
d.Health = "none"
return d
}
m := arr[0]

// Image (Config.Image)
if cfg, ok := m["Config"].(map[string]any); ok {
if img, ok := cfg["Image"].(string); ok {
d.Image = img
}
}

// Created
if created, ok := m["Created"].(string); ok {
d.CreatedAt = created
}

// State.*
if st, ok := m["State"].(map[string]any); ok {
if status, ok := st["Status"].(string); ok {
d.Status = status
}
if rc, ok := st["RestartCount"].(float64); ok {
d.RestartCount = int64(rc)
}
// Health
if h, ok := st["Health"].(map[string]any); ok {
if hs, ok := h["Status"].(string); ok {
d.Health = hs
}
} else {
d.Health = "none"
}
}

if d.Status == "" {
d.Status = "unknown"
}
if d.Health == "" {
d.Health = "none"
}
return d
}

func getHealth() (HealthResponse, error) {
host := readHostHealth()
wg := readDockerHealth("wg-easy")

resp := HealthResponse{Host: host, WGEasy: wg}
return resp, nil
}
