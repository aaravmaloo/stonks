CREATE INDEX IF NOT EXISTS idx_employee_candidates_season_id
ON game.employee_candidates (season_id, id);

CREATE INDEX IF NOT EXISTS idx_employee_candidates_best_value
ON game.employee_candidates (
    season_id,
    ((revenue_per_tick_micros::numeric / NULLIF(hire_cost_micros, 0))),
    risk_bps,
    id
);

CREATE INDEX IF NOT EXISTS idx_employee_candidates_high_output
ON game.employee_candidates (
    season_id,
    revenue_per_tick_micros DESC,
    risk_bps ASC,
    id ASC
);

CREATE INDEX IF NOT EXISTS idx_employee_candidates_low_risk
ON game.employee_candidates (
    season_id,
    risk_bps ASC,
    revenue_per_tick_micros DESC,
    id ASC
);

CREATE INDEX IF NOT EXISTS idx_business_employees_hire_lookup
ON game.business_employees (business_id, season_id, source_candidate_id);
