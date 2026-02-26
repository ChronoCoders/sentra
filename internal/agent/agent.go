package agent

import (
	"context"
	"time"

	"github.com/ChronoCoders/sentra/internal/models"
	"github.com/ChronoCoders/sentra/internal/wireguard"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type Agent struct {
	wg       wireguard.Manager
	reporter Reporter
	serverID string
}

func New(wg wireguard.Manager, reporter Reporter, serverID string) *Agent {
	return &Agent{wg: wg, reporter: reporter, serverID: serverID}
}

func (a *Agent) Run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var status *models.Status

			if a.wg != nil {
				s, err := a.wg.GetStatus(ctx)
				if err != nil {
					// Only log warning if it's NOT the "file does not exist" error which is expected in containers without WG
					errMsg := err.Error()
					if errMsg != "failed to get device wg0: file does not exist" && errMsg != "link not found" {
						log.Warn().Err(err).Msg("failed to get wireguard status (continuing with system metrics)")
					}
					status = &models.Status{}
				} else {
					status = s
				}
			} else {
				status = &models.Status{}
			}

			// Collect System Metrics
			sysInfo := a.collectSystemInfo()
			status.System = sysInfo

			event := models.StatusEvent{
				ServerID: a.serverID,
				Status:   status,
				Time:     time.Now(),
			}

			if err := a.reporter.Report(ctx, event); err != nil {
				log.Error().Err(err).Msg("failed to report status")
			} else {
				log.Info().Int("peer_count", len(status.Peers)).Msg("agent status reported")
			}
		}
	}
}

func (a *Agent) collectSystemInfo() models.SystemInfo {
	var info models.SystemInfo

	if h, err := host.Info(); err == nil {
		info.Hostname = h.Hostname
		info.OS = h.OS
		info.KernelVersion = h.KernelVersion
		info.Platform = h.Platform
		info.Uptime = h.Uptime
	}

	if c, err := cpu.Counts(true); err == nil {
		info.CPUCount = c
	}

	if p, err := cpu.Percent(0, false); err == nil && len(p) > 0 {
		info.CPUPercent = p[0]
	}

	if v, err := mem.VirtualMemory(); err == nil {
		info.MemoryTotal = v.Total
		info.MemoryUsed = v.Used
		info.MemoryPercent = v.UsedPercent
	}

	if d, err := disk.Usage("/"); err == nil {
		info.DiskTotal = d.Total
		info.DiskUsed = d.Used
		info.DiskPercent = d.UsedPercent
	}

	if l, err := load.Avg(); err == nil {
		info.LoadAverage = l.Load1
	}

	if n, err := net.IOCounters(false); err == nil && len(n) > 0 {
		info.NetBytesSent = n[0].BytesSent
		info.NetBytesRecv = n[0].BytesRecv
	}

	return info
}
