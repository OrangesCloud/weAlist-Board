package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestIncrementProjectCreated(t *testing.T) {
	m := getTestMetrics()
	
	// Get initial value
	initialValue := getCounterValue(t, m.ProjectCreatedTotal)

	// Increment
	m.IncrementProjectCreated()

	// Verify increment
	newValue := getCounterValue(t, m.ProjectCreatedTotal)
	if newValue <= initialValue {
		t.Errorf("Expected counter to increment, got %f -> %f", initialValue, newValue)
	}
}

func TestIncrementBoardCreated(t *testing.T) {
	m := getTestMetrics()
	
	// Get initial value
	initialValue := getCounterValue(t, m.BoardCreatedTotal)

	// Increment
	m.IncrementBoardCreated()

	// Verify increment
	newValue := getCounterValue(t, m.BoardCreatedTotal)
	if newValue <= initialValue {
		t.Errorf("Expected counter to increment, got %f -> %f", initialValue, newValue)
	}
}

func TestSetProjectsTotal(t *testing.T) {
	m := getTestMetrics()
	
	tests := []struct {
		name  string
		count int64
	}{
		{"zero projects", 0},
		{"one project", 1},
		{"multiple projects", 42},
		{"large number", 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.SetProjectsTotal(tt.count)
			value := getGaugeValue(t, m.ProjectsTotal)
			if value != float64(tt.count) {
				t.Errorf("Expected gauge value %d, got %f", tt.count, value)
			}
		})
	}
}

func TestSetBoardsTotal(t *testing.T) {
	m := getTestMetrics()
	
	tests := []struct {
		name  string
		count int64
	}{
		{"zero boards", 0},
		{"one board", 1},
		{"multiple boards", 100},
		{"large number", 5000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.SetBoardsTotal(tt.count)
			value := getGaugeValue(t, m.BoardsTotal)
			if value != float64(tt.count) {
				t.Errorf("Expected gauge value %d, got %f", tt.count, value)
			}
		})
	}
}

func TestBusinessMetricsIntegration(t *testing.T) {
	m := getTestMetrics()
	
	// Set initial totals
	m.SetProjectsTotal(10)
	m.SetBoardsTotal(50)

	// Verify initial values
	if getGaugeValue(t, m.ProjectsTotal) != 10 {
		t.Error("Expected ProjectsTotal to be 10")
	}
	if getGaugeValue(t, m.BoardsTotal) != 50 {
		t.Error("Expected BoardsTotal to be 50")
	}

	// Increment creation counters
	initialProjectCreated := getCounterValue(t, m.ProjectCreatedTotal)
	initialBoardCreated := getCounterValue(t, m.BoardCreatedTotal)

	m.IncrementProjectCreated()
	m.IncrementBoardCreated()
	m.IncrementBoardCreated()

	// Verify counters
	if getCounterValue(t, m.ProjectCreatedTotal) <= initialProjectCreated {
		t.Error("Expected ProjectCreatedTotal to increment")
	}
	if getCounterValue(t, m.BoardCreatedTotal) <= initialBoardCreated {
		t.Error("Expected BoardCreatedTotal to increment")
	}

	// Update totals
	m.SetProjectsTotal(11)
	m.SetBoardsTotal(52)

	// Verify updated values
	if getGaugeValue(t, m.ProjectsTotal) != 11 {
		t.Error("Expected ProjectsTotal to be 11")
	}
	if getGaugeValue(t, m.BoardsTotal) != 52 {
		t.Error("Expected BoardsTotal to be 52")
	}
}

// Helper function to get counter value
func getCounterValue(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		t.Fatalf("Failed to write counter metric: %v", err)
	}
	return metric.Counter.GetValue()
}

// Helper function to get gauge value
func getGaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		t.Fatalf("Failed to write gauge metric: %v", err)
	}
	return metric.Gauge.GetValue()
}
