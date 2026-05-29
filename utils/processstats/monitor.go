package processstats

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const defaultInterval = 1500 * time.Millisecond // igual que htop por defecto

// pidSnapshot guarda la última lectura de ticks de un proceso para calcular el delta.
type pidSnapshot struct {
	processTicks uint64
	systemTicks  uint64
}

// Monitor es un monitor de fondo que muestrea la CPU de una lista de PIDs cada intervalo
// y expone el último resultado disponible de forma instantánea (0ms de latencia en la API).
// Sigue el mismo modelo que htop: un goroutine de fondo actualiza la caché y la API solo lee.
type Monitor struct {
	mu       sync.RWMutex
	interval time.Duration
	prev     map[int]pidSnapshot
	results  map[int]InstanceStats
}

// NewMonitor crea un nuevo Monitor con el intervalo dado.
// Si interval es 0 se usa el valor por defecto de 1.5 segundos (igual que htop).
func NewMonitor(interval time.Duration) *Monitor {
	if interval == 0 {
		interval = defaultInterval
	}
	return &Monitor{
		interval: interval,
		prev:     make(map[int]pidSnapshot),
		results:  make(map[int]InstanceStats),
	}
}

// Start arranca el goroutine de fondo que actualiza las métricas periódicamente.
// Se detiene cuando el contexto es cancelado.
func (m *Monitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		slog.Info("CPU monitor started", "interval", m.interval)
		for {
			select {
			case <-ctx.Done():
				slog.Info("CPU monitor stopped")
				return
			case <-ticker.C:
				m.sample()
			}
		}
	}()
}

// sample realiza una ronda de muestreo para todos los PIDs registrados en el ciclo anterior.
// Los PIDs que ya no existen se limpian automáticamente.
func (m *Monitor) sample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	newPrev := make(map[int]pidSnapshot, len(m.prev))
	newResults := make(map[int]InstanceStats, len(m.prev))

	for pid, snap := range m.prev {
		currProcess, currSys, rssPages, ok := sampleForPid(pid)
		if !ok {
			// El proceso ya no existe, se limpia automáticamente (no se copia)
			continue
		}
		cpuPercent := calculatePercent(snap.processTicks, currProcess, snap.systemTicks, currSys)
		newResults[pid] = InstanceStats{
			CPUPercent:  cpuPercent,
			MemoryBytes: rssPages * uint64(pageSize()),
		}
		newPrev[pid] = pidSnapshot{processTicks: currProcess, systemTicks: currSys}
	}

	m.prev = newPrev
	m.results = newResults
}

// SyncPIDs actualiza el conjunto de PIDs monitoreados para que coincida exactamente
// con el mapa dado (serviceID -> PID). Los PIDs nuevos se registran y los que
// ya no están se eliminan. Es seguro llamarlo desde cualquier goroutine.
func (m *Monitor) SyncPIDs(active map[string]int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Construir conjunto de PIDs activos
	activePIDs := make(map[int]struct{}, len(active))
	for _, pid := range active {
		activePIDs[pid] = struct{}{}
	}

	// Eliminar PIDs que ya no están activos
	for pid := range m.prev {
		if _, ok := activePIDs[pid]; !ok {
			delete(m.prev, pid)
			delete(m.results, pid)
		}
	}

	// Registrar PIDs nuevos con una lectura base inicial
	for _, pid := range active {
		if pid <= 0 {
			continue
		}
		if _, exists := m.prev[pid]; !exists {
			process, sys, _, ok := sampleForPid(pid)
			if ok {
				m.prev[pid] = pidSnapshot{processTicks: process, systemTicks: sys}
			}
		}
	}
}

// Register añade un PID al conjunto de procesos monitoreados.
// Es seguro llamarlo desde cualquier goroutine.
func (m *Monitor) Register(pid int) {
	if pid <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.prev[pid]; exists {
		return
	}
	// Tomar una lectura inicial para tener base en el próximo ciclo
	process, sys, _, ok := sampleForPid(pid)
	if !ok {
		return
	}
	m.prev[pid] = pidSnapshot{processTicks: process, systemTicks: sys}
}

// Unregister elimina un PID del conjunto de procesos monitoreados.
func (m *Monitor) Unregister(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.prev, pid)
	delete(m.results, pid)
}

// Get devuelve el último InstanceStats calculado para un PID.
// Retorna instantáneamente (0ms). Si el PID aún no tiene datos, devuelve valores en 0.
func (m *Monitor) Get(pid int) InstanceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.results[pid]
}

// GetProcessStats es un alias de conveniencia para compatibilidad con el código anterior.
// Usa la instancia global del monitor para devolver la última lectura disponible.
func GetProcessStats(pid int) InstanceStats {
	return DefaultMonitor.Get(pid)
}

// DefaultMonitor es la instancia global del monitor, lista para usar directamente.
// Debe iniciarse con DefaultMonitor.Start(ctx) al arrancar la aplicación.
var DefaultMonitor = NewMonitor(0)
