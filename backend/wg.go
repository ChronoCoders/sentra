package main

import (
"bufio"
"os/exec"
"strconv"
"strings"
"time"

"github.com/rs/zerolog/log"
)

type Peer struct {
Key       string `json:"key"`
Endpoint  string `json:"endpoint"`
Handshake int64  `json:"handshake"`
RX        int64  `json:"rx"`
TX        int64  `json:"tx"`
}

type Status struct {
Interface string `json:"interface"`
Port      int    `json:"port"`
PublicIP  string `json:"public_ip"`
Peers     []Peer `json:"peers"`
}

const wgContainer = "wg-easy"
const wgIface = "wg0"

func getWGStatus() (Status, error) {
st := Status{
Interface: wgIface,
PublicIP:  getPublicIP(),
Peers:     []Peer{},
}

showOut, err := exec.Command("docker", "exec", wgContainer, "wg", "show", wgIface).Output()
if err != nil {
log.Error().Err(err).Str("container", wgContainer).Msg("Failed to execute wg show")
return st, err
}
parseWGShow(&st, string(showOut))

dumpOut, err := exec.Command("docker", "exec", wgContainer, "wg", "show", wgIface, "dump").Output()
if err != nil {
log.Warn().Err(err).Msg("Failed to execute wg dump, using show data only")
return st, nil
}
mergeWGDUMP(&st, string(dumpOut))

log.Debug().Int("peer_count", len(st.Peers)).Msg("WireGuard status retrieved")
return st, nil
}

func parseWGShow(st *Status, out string) {
sc := bufio.NewScanner(strings.NewReader(out))
var cur *Peer

for sc.Scan() {
line := strings.TrimSpace(sc.Text())
if line == "" {
continue
}

if strings.HasPrefix(line, "interface:") {
parts := strings.Fields(line)
if len(parts) >= 2 {
st.Interface = parts[1]
}
continue
}

if strings.HasPrefix(line, "listening port:") {
parts := strings.Fields(line)
if len(parts) >= 3 {
p, _ := strconv.Atoi(parts[2])
st.Port = p
}
continue
}

if strings.HasPrefix(line, "peer:") {
parts := strings.Fields(line)
if len(parts) >= 2 {
p := Peer{Key: parts[1]}
st.Peers = append(st.Peers, p)
cur = &st.Peers[len(st.Peers)-1]
}
continue
}

if cur == nil {
continue
}

if strings.HasPrefix(line, "endpoint:") {
cur.Endpoint = strings.TrimSpace(strings.TrimPrefix(line, "endpoint:"))
continue
}

if strings.HasPrefix(line, "latest handshake:") {
cur.Handshake = parseLooseHandshakeSeconds(line)
continue
}
}
}

func mergeWGDUMP(st *Status, dump string) {
raw := strings.TrimSpace(dump)
if raw == "" {
return
}
lines := strings.Split(raw, "\n")
if len(lines) == 0 {
return
}

now := time.Now().Unix()

index := map[string]*Peer{}
for i := range st.Peers {
index[st.Peers[i].Key] = &st.Peers[i]
}

for _, l := range lines[1:] {
f := strings.Fields(l)
if len(f) < 8 {
continue
}

key := f[0]
hsEpoch, _ := strconv.ParseInt(f[4], 10, 64)
rx, _ := strconv.ParseInt(f[5], 10, 64)
tx, _ := strconv.ParseInt(f[6], 10, 64)

p, ok := index[key]
if !ok {
st.Peers = append(st.Peers, Peer{Key: key})
p = &st.Peers[len(st.Peers)-1]
index[key] = p
}

p.RX = rx
p.TX = tx

if hsEpoch > 0 {
p.Handshake = max64(0, now-hsEpoch)
} else if p.Handshake < 0 {
p.Handshake = 0
}

if p.Endpoint == "" && f[2] != "(none)" {
p.Endpoint = f[2]
}
}
}

func parseLooseHandshakeSeconds(line string) int64 {
parts := strings.Fields(line)
var total int64 = 0
for i := 0; i < len(parts); i++ {
n, err := strconv.ParseInt(parts[i], 10, 64)
if err != nil || i+1 >= len(parts) {
continue
}
unit := parts[i+1]
switch {
case strings.HasPrefix(unit, "second"):
total += n
case strings.HasPrefix(unit, "minute"):
total += n * 60
case strings.HasPrefix(unit, "hour"):
total += n * 3600
case strings.HasPrefix(unit, "day"):
total += n * 86400
}
}
return total
}

func getPublicIP() string {
out, err := exec.Command("curl", "-s", "https://api.ipify.org").Output()
if err != nil {
log.Warn().Err(err).Msg("Failed to get public IP")
return "-"
}
ip := strings.TrimSpace(string(out))
if ip == "" {
return "-"
}
return ip
}

func max64(a, b int64) int64 {
if a > b {
return a
}
return b
}
