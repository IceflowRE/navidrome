package tests

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type MockChartRepo struct {
	MockedData []model.ChartEntry
}

func (m *MockChartRepo) GetChart(ctx context.Context, options model.ChartOptions) ([]model.ChartEntry, error) {
	return m.MockedData, nil
}

var _ model.ChartRepository = (*MockChartRepo)(nil)
