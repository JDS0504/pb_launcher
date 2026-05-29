package processstats

import (
	"context"
	"log/slog"
	"os"
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

// sample realiza una ronda de muestreo para todos los PIDs registrados.
// Los PIDs cuyo proceso ya no existe se limpian automáticamente.
func (m *Monitor) sample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	newPrev := make(map[int]pidSnapshot, len(m.prev))
	newResults := make(map[int]InstanceStats, len(m.prev))

	for pid, snap := range m.prev {
		currProcess, currSys, rssPages, ok := sampleForPid(pid)
		if !ok {
			continue // proceso muerto, se limpia solo al no copiarlo
		}
		cpuPercent := calculatePercent(snap.processTicks, currProcess, snap.systemTicks, currSys)
		newResults[pid] = InstanceStats{
			CPUPercent:  cpuPercent,
			MemoryBytes: rssPages * uint64(os.Getpagesize()),
		}
		newPrev[pid] = pidSnapshot{processTicks: currProcess, systemTicks: currSys}
	}

	m.prev = newPrev
	m.results = newResults
}

// Register añade un PID al conjunto de procesos monitoreados.
// Llamar al iniciar un servicio (event-driven, igual que htop al detectar un proceso nuevo).
func (m *Monitor) Register(pid int) {
	if pid <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.prev[pid]; exists {
		return
	}
	// Tomar lectura base inicial para calcular el delta en el próximo ciclo
	process, sys, _, ok := sampleForPid(pid)
	if !ok {
		return
	}
	m.prev[pid] = pidSnapshot{processTicks: process, systemTicks: sys}
}

// Unregister elimina un PID del conjunto de procesos monitoreados.
// Llamar al detener o suspender un servicio.
func (m *Monitor) Unregister(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.prev, pid)
	delete(m.results, pid)
}

// Get devuelve el último InstanceStats calculado para un PID.
// Retorna instantáneamente (0ms de latencia).
func (m *Monitor) Get(pid int) InstanceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.results[pid]
}

// DefaultMonitor es la instancia global del monitor.
// Debe iniciarse con DefaultMonitor.Start(ctx) al arrancar la aplicación.
var DefaultMonitor = NewMonitor(0)
