# Meta-Repo Strategy Workflow

This document describes the strategy for using DepViz with a meta-repository approach,
where a single repository tracks cross-project dependencies and priorities.

## Overview

A meta-repo board aggregates work items from multiple source repositories into a unified
dependency graph. This enables:

- Cross-repo dependency tracking
- Unified priority views across teams
- Automated freshness via GitHub App webhooks
- Shared and personal view overlays

## Setup

1. Create a dedicated repository (e.g., `org/meta`) for tracking cross-project work
2. Configure DepViz with GitHub App auth for webhook-driven freshness
3. Add source repositories via the workspace panel
4. Define dependency links between items across repos

## Board Scoping

Use scope queries to filter the meta-board view:
- `org:myorg` — all items from an organization
- `repo:owner/repo` — items from a specific repository
- Empty scope — all tracked items

## Shared vs Personal Views

Board views can be:
- **Personal** — visible only to you, stored per-account
- **Shared** — visible to all authenticated users on this DepViz instance

## Automation

With GitHub App webhooks configured, DepViz automatically:
- Updates node state when issues/PRs are opened, closed, or merged
- Refreshes freshness scores in real time
- Notifies the board of changes without manual sync

## Multi-Workspace Support

Use the workspace switcher to scope the graph to a specific org, repo, or project.
Switch between workspaces without losing your personal overrides or saved views.
