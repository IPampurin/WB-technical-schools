package collector

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
)

// MetricsJSON структура ответа для /api/stats
type MetricsJSON struct {
	AllocBytes      string  `json:"alloc_bytes"`       // используемая память (человеко‑читаемый вид)
	TotalAllocBytes string  `json:"total_alloc_bytes"` // всего выделено за всё время
	Mallocs         uint64  `json:"mallocs"`           // количество аллокаций (выделений памяти) – добавлено
	NumGC           uint32  `json:"num_gc"`            // количество сборок GC
	LastGCTime      string  `json:"last_gc_time"`      // время последнего GC в формате "2006-01-02 15:04:05.000"
	GCCPUFraction   float64 `json:"gc_cpu_fraction"`   // доля CPU на GC
	GCPercents      int     `json:"gc_percent"`        // текущее значение GOGC
}

// GetCurrentMetrics собирает свежие метрики из runtime.MemStats и GOGC
func GetCurrentMetrics() MetricsJSON {

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// получаем GOGC без изменения
	gcPercent := debug.SetGCPercent(-1)
	debug.SetGCPercent(gcPercent)

	// форматируем время последней сборки мусора
	lastGCTimeStr := "ещё не было"
	if ms.LastGC != 0 {
		t := time.Unix(0, int64(ms.LastGC))
		lastGCTimeStr = t.Format("2006-01-02 15:04:05.000")
	}

	return MetricsJSON{
		AllocBytes:      formatBytes(ms.Alloc),
		TotalAllocBytes: formatBytes(ms.TotalAlloc),
		Mallocs:         ms.Mallocs,
		NumGC:           ms.NumGC,
		LastGCTime:      lastGCTimeStr,
		GCCPUFraction:   ms.GCCPUFraction,
		GCPercents:      gcPercent,
	}
}

// formatBytes переводит байты в человеко‑читаемый вид (KB, MB, GB)
func formatBytes(b uint64) string {

	const unit = 1024
	if b < unit {
		return strconv.FormatUint(b, 10) + " B"
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return strconv.FormatFloat(float64(b)/float64(div), 'f', 2, 64) + " " + []string{"KB", "MB", "GB", "TB"}[exp] + "B"
}
