package systemstatus

import (
	"net/http"
	"os/exec"
	launcherdomain "pb_launcher/internal/launcher/domain"
	"runtime"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// getProcessStats obtiene el % de CPU y RAM real (RSS) de un PID usando "ps" en Linux
func getProcessStats(pid int) InstanceStats {
	if pid <= 0 {
		return InstanceStats{}
	}
	if runtime.GOOS == "windows" {
		return InstanceStats{}
	}
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu,rss")
	out, err := cmd.Output()
	if err != nil {
		return InstanceStats{}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return InstanceStats{}
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return InstanceStats{}
	}
	cpu, _ := strconv.ParseFloat(fields[0], 64)
	rssKb, _ := strconv.ParseUint(fields[1], 10, 64)
	return InstanceStats{
		CPUPercent:  cpu,
		MemoryBytes: rssKb * 1024, // Convertir de KB a bytes
	}
}

// RegisterRoutes registers the /x-api/system/status route in PocketBase
func RegisterRoutes(app *pocketbase.PocketBase, launcherManager *launcherdomain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			// Get number of active instances running currently in memory
			activeInstances := launcherManager.GetActiveInstancesCount()

			// Collect system metrics with "." representing the primary storage partition
			status, err := CollectStatus(".")
			if err != nil {
				return e.BadRequestError("failed to collect system status", err)
			}

			// Populate dynamic metadata
			status.Host.ActiveInstances = activeInstances

			// Collect stats for each running instance process
			status.InstancesStats = make(map[string]InstanceStats)
			for id, pid := range launcherManager.GetRunningInstancesPIDs() {
				status.InstancesStats[id] = getProcessStats(pid)
			}

			return e.JSON(http.StatusOK, status)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
