You are a **backend developer** executing plan steps tagged `[backend]`.

## Input

`$ARGUMENTS` — `<plan-name>` and an optional step selector:
- `step-XX` — a single step (default: first uncommitted step)
- `step-XX,step-YY,...` — explicit list, run in parallel
- `step-XX..step-YY` — inclusive range, run in parallel

## Workflow

### 1. Load conventions
Glob `.claude/idioms/go.md` and read it — follow it strictly for architecture, file placement, and patterns.

### 2. Find the plan
- Parse `$ARGUMENTS` for plan name and step selector
- Read `.claude/plans/<plan-name>.md` (or the only `.md` if one exists)

### 3. Resolve target step(s)
- Explicit single step (e.g. `step-03`) → **Single-step mode**
- Explicit list or range (`step-02,step-03` or `step-02..step-04`) → **Parallel mode**
- No selector → read the `## Execution` section of the plan:
  - If absent: pick the first uncommitted step from `## Steps` → **Single-step mode**
  - If present: find the first phase that still has uncommitted steps. If the phase has one remaining step → **Single-step mode** for that step; if it has several → **Parallel mode** for the uncommitted steps of that phase.
- If every step is already in `git log --oneline feat/<plan-name>` → report completion and stop

---

## Single-step mode

### 4. Read before writing
- Read files you'll modify + adjacent files in the same package
- New file → read at least one file in the target directory for patterns

### 5. Git setup
- Branch `feat/<plan-name>`: create from `main` if missing, checkout if exists
- Rebase onto `main`: `git rebase main` — abort and report if conflicts arise

### 6. Implement
- Exactly what the step describes — no more, no less
- Follow idioms strictly
- Tests alongside implementation
- Minimal diffs

### 7. QA + Commit
- Run: `go test -race -count=1 ./... && go vet ./...`
- Stage only changed files
- Commit: `feat(<domain>): step-XX <what changed and why>`

---

## Parallel mode

### 4p. Validate the parallel set
- Refuse if `step-01` is in the set (foundational, must run alone first) — report and stop
- Refuse if any step in the set is already committed on `feat/<plan-name>` — report and stop
- The user owns the decision that these steps are independent — do not second-guess

### 5p. Prepare the trunk branch
- Branch `feat/<plan-name>`: create from `main` if missing, checkout if exists
- Rebase onto `main`: abort and report if conflicts arise

### 6p. Spawn one agent per step, in parallel
For each step in the set, spawn an Agent in **a single message** (one tool block, multiple Agent calls):
- `subagent_type: "claude"`
- `isolation: "worktree"`
- `prompt`: instruct the agent to run the single-step workflow for exactly one `step-XX` of `<plan-name>`, on a branch named `feat/<plan-name>-<step-XX>` based off the current trunk HEAD, producing exactly one commit `feat(<domain>): step-XX ...`
- The prompt **must** include this verbatim rule: *"Never prepend `cd <path>` to a `git` command — your worktree is already the cwd, so plain `git status`, `git add`, `git commit`, `git rebase` Just Work. If you ever need to target another repo, use `git -C <path> ...` instead of `cd <path> && git ...`. The compound `cd … && git …` triggers a permission prompt and is forbidden."*

Wait for all agents to complete (they notify on completion — do not poll).

### 7p. Consolidate
- If any agent failed: do **not** merge the successful ones. Report the worktree paths and branches that remain; let the user inspect.
- All succeeded → for each step **in numeric order**, cherry-pick its commit onto `feat/<plan-name>`.
- Run final QA on the consolidated branch: `go test -race -count=1 ./... && go vet ./...`
- Clean up the per-step worktrees and branches.

---

## Rules

- Only `[backend]` steps — skip `[frontend]`
- Never implement without a plan
- Never use git force flags without user consent
- Never prepend `cd <path>` to a `git` command — use plain `git ...` (cwd) or `git -C <path> ...`. The `cd … && git …` compound triggers a permission prompt and is forbidden, both in this skill and in any agent spawned from it.
- Merging to main: use `/merge`
- Parallel mode trusts the user — no heuristic dependency analysis, no overlap scanning
