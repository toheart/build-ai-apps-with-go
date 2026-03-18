package memory

import (
	"context"
	"sync"
	"time"

	domainsample "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/sample"
)

var _ domainsample.Repository = (*SampleRepository)(nil)

type SampleRepository struct {
	mu      sync.RWMutex
	samples []domainsample.Sample
}

func NewSampleRepository() *SampleRepository {
	now := time.Now().UTC()

	return &SampleRepository{
		samples: []domainsample.Sample{
			{
				ID:        "sample-001",
				Name:      "Backend layering",
				Summary:   "Shows how the Go service separates domain, application, and HTTP delivery.",
				Category:  domainsample.CategoryBackend,
				Status:    domainsample.StatusReady,
				UpdatedAt: now.Add(-15 * time.Minute),
			},
			{
				ID:        "sample-002",
				Name:      "Frontend request flow",
				Summary:   "Demonstrates typed API access, loading state, and reusable page components.",
				Category:  domainsample.CategoryFrontend,
				Status:    domainsample.StatusInProgress,
				UpdatedAt: now.Add(-45 * time.Minute),
			},
			{
				ID:        "sample-003",
				Name:      "Project conventions",
				Summary:   "Captures API, testing, and coding conventions so later AI work has a stable base.",
				Category:  domainsample.CategoryWorkflow,
				Status:    domainsample.StatusDone,
				UpdatedAt: now.Add(-2 * time.Hour),
			},
		},
	}
}

func (r *SampleRepository) List(ctx context.Context) ([]domainsample.Sample, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]domainsample.Sample, len(r.samples))
	copy(items, r.samples)

	return items, nil
}
