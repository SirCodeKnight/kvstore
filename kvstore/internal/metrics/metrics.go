package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics represents the metrics collection for the key-value store
type Metrics struct {
	// Counters
	gets        prometheus.Counter
	sets        prometheus.Counter
	deletes     prometheus.Counter
	getHits     prometheus.Counter
	getMisses   prometheus.Counter
	raftApplies prometheus.Counter
	
	// Histograms
	getLatency    prometheus.Histogram
	setLatency    prometheus.Histogram
	deleteLatency prometheus.Histogram
	
	// Gauges
	clusterSize   prometheus.Gauge
	isLeader      prometheus.Gauge
	keysCount     prometheus.Gauge
	bytesStored   prometheus.Gauge
}

// NewMetrics creates a new metrics collection
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// Counters
		gets: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "gets_total",
			Help:      "Total number of GET operations",
		}),
		
		sets: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sets_total",
			Help:      "Total number of SET operations",
		}),
		
		deletes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "deletes_total",
			Help:      "Total number of DELETE operations",
		}),
		
		getHits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "get_hits_total",
			Help:      "Total number of GET operations that found the key",
		}),
		
		getMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "get_misses_total",
			Help:      "Total number of GET operations that did not find the key",
		}),
		
		raftApplies: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "raft_applies_total",
			Help:      "Total number of Raft log entries applied",
		}),
		
		// Histograms
		getLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "get_latency_seconds",
			Help:      "Latency of GET operations in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 16),
		}),
		
		setLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "set_latency_seconds",
			Help:      "Latency of SET operations in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 16),
		}),
		
		deleteLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "delete_latency_seconds",
			Help:      "Latency of DELETE operations in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 16),
		}),
		
		// Gauges
		clusterSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cluster_size",
			Help:      "Number of nodes in the cluster",
		}),
		
		isLeader: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "is_leader",
			Help:      "Whether this node is the leader (1) or not (0)",
		}),
		
		keysCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "keys_count",
			Help:      "Number of keys in the store",
		}),
		
		bytesStored: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "bytes_stored",
			Help:      "Total bytes stored",
		}),
	}
	
	// Register the metrics
	prometheus.MustRegister(
		m.gets,
		m.sets,
		m.deletes,
		m.getHits,
		m.getMisses,
		m.raftApplies,
		m.getLatency,
		m.setLatency,
		m.deleteLatency,
		m.clusterSize,
		m.isLeader,
		m.keysCount,
		m.bytesStored,
	)
	
	return m
}

// IncGet increments the get counter
func (m *Metrics) IncGet() {
	m.gets.Inc()
}

// IncSet increments the set counter
func (m *Metrics) IncSet() {
	m.sets.Inc()
}

// IncDelete increments the delete counter
func (m *Metrics) IncDelete() {
	m.deletes.Inc()
}

// IncGetHit increments the get hit counter
func (m *Metrics) IncGetHit() {
	m.getHits.Inc()
}

// IncGetMiss increments the get miss counter
func (m *Metrics) IncGetMiss() {
	m.getMisses.Inc()
}

// IncRaftApply increments the Raft apply counter
func (m *Metrics) IncRaftApply() {
	m.raftApplies.Inc()
}

// ObserveGetLatency observes a GET latency
func (m *Metrics) ObserveGetLatency(seconds float64) {
	m.getLatency.Observe(seconds)
}

// ObserveSetLatency observes a SET latency
func (m *Metrics) ObserveSetLatency(seconds float64) {
	m.setLatency.Observe(seconds)
}

// ObserveDeleteLatency observes a DELETE latency
func (m *Metrics) ObserveDeleteLatency(seconds float64) {
	m.deleteLatency.Observe(seconds)
}

// SetClusterSize sets the cluster size gauge
func (m *Metrics) SetClusterSize(size int) {
	m.clusterSize.Set(float64(size))
}

// SetIsLeader sets the is_leader gauge (1 if leader, 0 if not)
func (m *Metrics) SetIsLeader(isLeader bool) {
	if isLeader {
		m.isLeader.Set(1)
	} else {
		m.isLeader.Set(0)
	}
}

// SetKeysCount sets the keys count gauge
func (m *Metrics) SetKeysCount(count int) {
	m.keysCount.Set(float64(count))
}

// AddBytesStored adds to the bytes stored gauge
func (m *Metrics) AddBytesStored(bytes int) {
	m.bytesStored.Add(float64(bytes))
}

// SubBytesStored subtracts from the bytes stored gauge
func (m *Metrics) SubBytesStored(bytes int) {
	m.bytesStored.Sub(float64(bytes))
}