package database

import (
	"context"
	"fmt"
	"time"
)

type HuntMetricRecord struct {
	HuntID             string     `json:"hunt_id"`
	Name               string     `json:"name"`
	Version            int        `json:"version"`
	RunCount           int64      `json:"run_count"`
	TotalResultCount   int64      `json:"total_result_count"`
	AverageResultCount float64    `json:"average_result_count"`
	LastRunAt          *time.Time `json:"last_run_at,omitempty"`
}

type HuntMetricsSummary struct {
	HuntCount    int64              `json:"hunt_count"`
	TotalRuns    int64              `json:"total_runs"`
	TotalResults int64              `json:"total_results"`
	Hunts        []HuntMetricRecord `json:"hunts"`
}

type InvestigationMetricsSummary struct {
	Total         int64            `json:"total"`
	Unassigned    int64            `json:"unassigned"`
	ByStatus      map[string]int64 `json:"by_status"`
	ByPriority    map[string]int64 `json:"by_priority"`
	OldestOpenAt  *time.Time       `json:"oldest_open_at,omitempty"`
	LastUpdatedAt *time.Time       `json:"last_updated_at,omitempty"`
}

func (d *Database) HuntMetrics(ctx context.Context) (HuntMetricsSummary, error) {
	if d == nil || d.pool == nil {
		return HuntMetricsSummary{}, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			h.id,
			h.name,
			h.current_version,
			COUNT(hr.id) AS run_count,
			COALESCE(SUM(hr.result_count), 0) AS total_result_count,
			COALESCE(AVG(hr.result_count), 0)::float8 AS average_result_count,
			MAX(hr.completed_at) AS last_run_at
		FROM hunts h
		LEFT JOIN hunt_runs hr ON hr.hunt_id = h.id
		GROUP BY h.id, h.name, h.current_version
		ORDER BY h.updated_at DESC, h.id
	`)
	if err != nil {
		return HuntMetricsSummary{}, fmt.Errorf("query hunt metrics: %w", err)
	}
	defer rows.Close()

	summary := HuntMetricsSummary{
		Hunts: make([]HuntMetricRecord, 0),
	}
	for rows.Next() {
		var metric HuntMetricRecord
		if err := rows.Scan(
			&metric.HuntID,
			&metric.Name,
			&metric.Version,
			&metric.RunCount,
			&metric.TotalResultCount,
			&metric.AverageResultCount,
			&metric.LastRunAt,
		); err != nil {
			return HuntMetricsSummary{}, fmt.Errorf("scan hunt metrics: %w", err)
		}
		summary.HuntCount++
		summary.TotalRuns += metric.RunCount
		summary.TotalResults += metric.TotalResultCount
		summary.Hunts = append(summary.Hunts, metric)
	}
	if err := rows.Err(); err != nil {
		return HuntMetricsSummary{}, fmt.Errorf("iterate hunt metrics: %w", err)
	}

	return summary, nil
}

func (d *Database) InvestigationMetrics(ctx context.Context) (InvestigationMetricsSummary, error) {
	if d == nil || d.pool == nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("database is not initialised")
	}

	summary := InvestigationMetricsSummary{
		ByStatus:   map[string]int64{},
		ByPriority: map[string]int64{},
	}

	if err := d.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE owner_id IS NULL),
			MIN(created_at) FILTER (WHERE status <> 'closed'),
			MAX(updated_at)
		FROM investigations
	`).Scan(
		&summary.Total,
		&summary.Unassigned,
		&summary.OldestOpenAt,
		&summary.LastUpdatedAt,
	); err != nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("query investigation summary: %w", err)
	}

	statusRows, err := d.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM investigations
		GROUP BY status
		ORDER BY status
	`)
	if err != nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("query investigation status metrics: %w", err)
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var (
			status string
			count  int64
		)
		if err := statusRows.Scan(&status, &count); err != nil {
			return InvestigationMetricsSummary{}, fmt.Errorf("scan investigation status metric: %w", err)
		}
		summary.ByStatus[status] = count
	}
	if err := statusRows.Err(); err != nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("iterate investigation status metrics: %w", err)
	}

	priorityRows, err := d.pool.Query(ctx, `
		SELECT priority, COUNT(*)
		FROM investigations
		GROUP BY priority
		ORDER BY priority
	`)
	if err != nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("query investigation priority metrics: %w", err)
	}
	defer priorityRows.Close()
	for priorityRows.Next() {
		var (
			priority string
			count    int64
		)
		if err := priorityRows.Scan(&priority, &count); err != nil {
			return InvestigationMetricsSummary{}, fmt.Errorf("scan investigation priority metric: %w", err)
		}
		summary.ByPriority[priority] = count
	}
	if err := priorityRows.Err(); err != nil {
		return InvestigationMetricsSummary{}, fmt.Errorf("iterate investigation priority metrics: %w", err)
	}

	return summary, nil
}
