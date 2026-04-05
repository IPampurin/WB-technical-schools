package collector

import (
	"runtime"
	"runtime/debug"

	"github.com/prometheus/client_golang/prometheus"
)

// MemStatsCollector реализует интерфейс prometheus.Collector
type MemStatsCollector struct {
	allocBytes      *prometheus.Desc
	totalAllocBytes *prometheus.Desc
	mallocs         *prometheus.Desc
	numGC           *prometheus.Desc
	lastGCTime      *prometheus.Desc
	gcCPUFraction   *prometheus.Desc
	gcPercent       *prometheus.Desc
}

// NewMemStatsCollector создаёт новый коллектор метрик Go runtime
func NewMemStatsCollector() *MemStatsCollector {
	return &MemStatsCollector{

		allocBytes: prometheus.NewDesc(
			"go_memstats_alloc_bytes",
			"Количество байт, выделенных и используемых в данный момент (HeapAlloc)",
			nil, nil,
		),
		totalAllocBytes: prometheus.NewDesc(
			"go_memstats_total_alloc_bytes",
			"Общее количество выделенных байт за всё время (TotalAlloc)",
			nil, nil,
		),
		mallocs: prometheus.NewDesc(
			"go_memstats_mallocs_total",
			"Общее количество выполненных аллокаций (выделений памяти)",
			nil, nil,
		),
		numGC: prometheus.NewDesc(
			"go_gc_num_gc",
			"Количество завершённых циклов GC",
			nil, nil,
		),
		lastGCTime: prometheus.NewDesc(
			"go_gc_last_gc_time_seconds",
			"Время последнего GC в секундах с эпохи Unix",
			nil, nil,
		),
		gcCPUFraction: prometheus.NewDesc(
			"go_gc_cpu_fraction",
			"Доля CPU, потраченная на GC с момента запуска программы",
			nil, nil,
		),
		gcPercent: prometheus.NewDesc(
			"go_gc_percent",
			"Текущее значение GOGC (процент)",
			nil, nil,
		),
	}
}

// Describe отправляет описания метрик в канал
func (c *MemStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.allocBytes
	ch <- c.totalAllocBytes
	ch <- c.mallocs
	ch <- c.numGC
	ch <- c.lastGCTime
	ch <- c.gcCPUFraction
	ch <- c.gcPercent
}

// Collect собирает актуальные значения метрик и отправляет их в канал
func (c *MemStatsCollector) Collect(ch chan<- prometheus.Metric) {

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// получаем GOGC
	gcPercent := debug.SetGCPercent(-1)
	debug.SetGCPercent(gcPercent)

	ch <- prometheus.MustNewConstMetric(c.allocBytes, prometheus.GaugeValue, float64(ms.Alloc))
	ch <- prometheus.MustNewConstMetric(c.totalAllocBytes, prometheus.CounterValue, float64(ms.TotalAlloc))
	ch <- prometheus.MustNewConstMetric(c.mallocs, prometheus.CounterValue, float64(ms.Mallocs))
	ch <- prometheus.MustNewConstMetric(c.numGC, prometheus.CounterValue, float64(ms.NumGC))
	ch <- prometheus.MustNewConstMetric(c.lastGCTime, prometheus.GaugeValue, float64(ms.LastGC)/1e9)
	ch <- prometheus.MustNewConstMetric(c.gcCPUFraction, prometheus.GaugeValue, ms.GCCPUFraction)
	ch <- prometheus.MustNewConstMetric(c.gcPercent, prometheus.GaugeValue, float64(gcPercent))
}
