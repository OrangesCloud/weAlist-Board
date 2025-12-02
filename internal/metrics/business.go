package metrics

// IncrementProjectCreated increments project creation counter
func (m *Metrics) IncrementProjectCreated() {
	m.safeExecute("IncrementProjectCreated", func() {
		m.ProjectCreatedTotal.Inc()
	})
}

// IncrementBoardCreated increments board creation counter
func (m *Metrics) IncrementBoardCreated() {
	m.safeExecute("IncrementBoardCreated", func() {
		m.BoardCreatedTotal.Inc()
	})
}

// SetProjectsTotal sets total projects gauge
func (m *Metrics) SetProjectsTotal(count int64) {
	m.safeExecute("SetProjectsTotal", func() {
		m.ProjectsTotal.Set(float64(count))
	})
}

// SetBoardsTotal sets total boards gauge
func (m *Metrics) SetBoardsTotal(count int64) {
	m.safeExecute("SetBoardsTotal", func() {
		m.BoardsTotal.Set(float64(count))
	})
}
