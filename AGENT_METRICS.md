# Agent Effectiveness Metrics

Track Athena Framework effectiveness with a layered scorecard.

## Scorecard Dimensions

1. Throughput: task completion without manual rescue.
2. Quality: defect escape after merge.
3. Safety: policy/secret/security violations.
4. Efficiency: cycle time from request to validated change.
5. Maintainability: tests/docs included when required.

## Suggested KPIs

- `ai_check_pass_rate`
- `first_pass_validation_rate`
- `agent_rework_rate`
- `security_block_rate`
- `doc_promotion_latency_days`

## Cadence

1. Baseline for 2-4 weeks.
2. Change one Athena Framework variable at a time.
3. Compare against baseline by task class.
4. Keep only changes with measurable improvement.
