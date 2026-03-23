# Scheduler

> Built-in cron-style task automation. No external daemon needed.

## Overview

BlackCat includes an in-process scheduler that runs as a goroutine alongside the channel gateway. Tasks persist in SQLite and execute automatically when you run `blackcat serve`. There is no external cron daemon or systemd timer to configure — everything lives inside the single BlackCat binary.

When a scheduled task fires, the scheduler creates a fresh agent session, injects the task prompt, and runs it through the full ReAct loop (reasoning, tool use, memory). The result can be routed to a channel, saved to a file, or both.

## Configuration

Add schedules to your `~/.blackcat/config.yaml`:

```yaml
scheduler:
  enabled: true
  schedules:
    - name: "daily-security-scan"
      cron: "0 9 * * *"           # Every day at 9 AM
      task: "Scan this repo for secrets and vulnerabilities, generate a report"

    - name: "weekly-dependency-audit"
      cron: "0 10 * * 1"          # Every Monday at 10 AM
      task: "Check all dependencies for known CVEs, prioritize by EPSS score"

    - name: "nightly-backup-check"
      cron: "0 2 * * *"           # Every day at 2 AM
      task: "Verify that all database backups completed successfully"

    - name: "monthly-architecture-review"
      cron: "0 14 1 * *"          # 1st of every month at 2 PM
      task: "Review this project against WAF 6 pillars, generate report"
```

## Slash Commands

Manage schedules at runtime without editing config files:

```
/schedule list                                    # Show all scheduled tasks
/schedule add "0 9 * * *" "scan for secrets"      # Add new schedule
/schedule remove daily-security-scan              # Remove by name
/schedule history                                 # View run history
```

## Cron Syntax

Standard 5-field cron expressions are supported:

| Field | Values | Example |
|-------|--------|---------|
| Minute | 0-59 | `30` (at minute 30) |
| Hour | 0-23 | `9` (at 9 AM) |
| Day of Month | 1-31 | `1` (1st of month) |
| Month | 1-12 | `*` (every month) |
| Day of Week | 0-6 (Sun=0) | `1` (Monday) |

Special characters:

| Character | Meaning | Example |
|-----------|---------|---------|
| `*` | Every value | `* * * * *` (every minute) |
| `,` | List | `1,3,5` (Mon, Wed, Fri) |
| `-` | Range | `9-17` (9 AM to 5 PM) |
| `/` | Step | `*/15` (every 15 minutes) |

## Output Routing

Schedule results can be routed to channels, files, or both:

```yaml
schedules:
  - name: "daily-scan"
    cron: "0 9 * * *"
    task: "Scan for vulnerabilities"
    output:
      channel: telegram      # Send result to Telegram
      # or: discord, slack, whatsapp, email

  - name: "weekly-report"
    cron: "0 10 * * 1"
    task: "Generate weekly security summary"
    output:
      file: "reports/weekly-security-{{date}}.md"

  - name: "compliance-check"
    cron: "0 8 * * *"
    task: "Run SOC2 compliance gap analysis"
    output:
      channel: slack
      file: "reports/compliance-{{date}}.md"   # Both channel and file
```

When no `output` is specified, results are stored in the scheduler history and can be viewed with `/schedule history`.

## Running

The scheduler runs automatically as part of `blackcat serve`:

```bash
blackcat serve   # Starts gateway (channels) + scheduler
```

The scheduler:
1. Loads all schedules from config and SQLite
2. Starts a goroutine with a cron ticker
3. Fires tasks at the scheduled times
4. Creates a fresh agent session per task execution
5. Routes output according to the `output` configuration
6. Records execution history (status, duration, result summary) in SQLite

## DevSecOps Examples

Real-world schedules for security and operations teams:

```yaml
schedules:
  # Daily secret scan across all repositories
  - name: "daily-secret-scan"
    cron: "0 6 * * *"
    task: "Scan this repo for hardcoded secrets using all 16 Gitleaks rules. Report any new findings since yesterday."
    output:
      channel: slack

  # Weekly dependency vulnerability audit
  - name: "weekly-cve-audit"
    cron: "0 9 * * 1"
    task: "Check all dependencies for known CVEs. Prioritize by EPSS score and KEV status. Generate a summary with remediation steps."
    output:
      channel: telegram
      file: "reports/cve-audit-{{date}}.md"

  # Nightly Dockerfile and IaC compliance check
  - name: "nightly-iac-scan"
    cron: "0 3 * * *"
    task: "Scan all Dockerfiles and Terraform files for security issues. Check against CIS benchmarks."
    output:
      file: "reports/iac-scan-{{date}}.md"

  # Bi-weekly SBOM generation
  - name: "biweekly-sbom"
    cron: "0 10 1,15 * *"
    task: "Generate a CycloneDX SBOM for this project. Compare with the previous SBOM and flag new dependencies."
    output:
      file: "reports/sbom-{{date}}.json"

  # Monthly SOC2 compliance gap analysis
  - name: "monthly-soc2-review"
    cron: "0 14 1 * *"
    task: "Run a SOC2 compliance gap analysis. Map current controls, identify gaps, and generate an executive summary."
    output:
      channel: slack
      file: "reports/soc2-gap-{{date}}.md"

  # CI/CD pipeline hardening audit every Monday
  - name: "weekly-pipeline-audit"
    cron: "0 8 * * 1"
    task: "Audit all GitHub Actions workflows. Check for mutable tags, overly permissive permissions, and secret handling issues. Generate hardened versions if needed."
    output:
      channel: slack
```

## Architecture Examples

Schedules for architecture review and capacity planning:

```yaml
schedules:
  # Monthly capacity projection
  - name: "monthly-capacity-check"
    cron: "0 10 1 * *"
    task: "Review current resource utilization. Project capacity needs for the next 3 months based on growth trends. Flag any services approaching limits."
    output:
      channel: slack
      file: "reports/capacity-{{date}}.md"

  # Quarterly cloud cost review
  - name: "quarterly-cost-review"
    cron: "0 14 1 1,4,7,10 *"
    task: "Analyze cloud infrastructure costs. Identify waste (idle resources, oversized instances, unused storage). Recommend right-sizing and reserved instance opportunities."
    output:
      file: "reports/cost-review-{{date}}.md"

  # Weekly architecture drift detection
  - name: "weekly-arch-drift"
    cron: "0 11 * * 5"
    task: "Compare current codebase structure against documented architecture. Flag any drift or new dependencies that were not in the original design."
    output:
      channel: discord

  # Monthly WAF (Well-Architected Framework) review
  - name: "monthly-waf-review"
    cron: "0 14 15 * *"
    task: "Review this project against all 6 WAF pillars (security, reliability, performance, cost, operations, sustainability). Score each pillar and recommend improvements."
    output:
      channel: slack
      file: "reports/waf-review-{{date}}.md"
```

## Troubleshooting

### Scheduled task not running

1. Verify `scheduler.enabled: true` in config
2. Ensure `blackcat serve` is running (scheduler only runs in serve mode)
3. Check cron syntax with `/schedule list` to see next fire time
4. Check `/schedule history` for past execution errors

### Task running but no output

1. Verify the `output` section is correctly configured
2. Check that the target channel is enabled and connected
3. Check file path permissions if using `file` output

### Task takes too long

Each scheduled task has a default timeout (configurable). Long-running tasks can be split into smaller, more focused schedules. The agent session created for each task has the same tool access and memory as an interactive session.
