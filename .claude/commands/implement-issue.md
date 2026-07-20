# Implement Single Linear Issue

Implement a specific Linear issue for mcp-for-argo-workflows.

## Usage

Provide the issue identifier (e.g., PIP-15) as an argument: `/implement-issue PIP-15`

The argument is: $ARGUMENTS

## Step 1: Fetch Issue Details

1. Use `mcp__linear-server__get_issue` with the provided issue ID
2. Parse the issue description for:
   - Tasks/requirements
   - Tool schema (if MCP tool)
   - Implementation notes
   - Dependencies
   - Acceptance criteria

## Step 2: Check Dependencies

1. Identify dependencies from the issue description (usually listed at bottom)
2. Verify each dependency is complete:
   - Check Linear status
   - Verify code exists locally
3. If dependencies not met, report and stop

## Step 3: Create Feature Branch

Create a branch for this issue using the Linear-suggested branch name:

```bash
git checkout main
git pull origin main
git checkout -b <branch-name-from-linear>
```

The branch name is provided in the Linear issue details as `gitBranchName` (e.g., `alan/pip-10-implement-mcp-server-skeleton`).

## Step 4: Update Linear Status

Move issue to "In Progress":
```
mcp__linear-server__update_issue(id: "<issue-id>", state: "In Progress")
```

## Step 5: Plan Agent Collaboration

Based on issue labels and content, determine which agents need to be involved:

### Primary Implementation Agent

| Label | Primary Agent |
|-------|---------------|
| `setup` | `go-developer` or `ci-devops` |
| `mcp-tool` | `mcp-tool-implementer` |
| `testing` | `testing` |
| `docs` | `docs-examples` |
| `ci` | `ci-devops` |

### Supporting Agents (as needed)

- **`testing`** - Write or update tests for the implementation
- **`docs-examples`** - Update README, CLAUDE.md, or add examples
- **`go-developer`** - Review Go code patterns and architecture
- **`kubernetes-argo`** - Review Argo/K8s integration code

### Agent Collaboration Patterns

1. **Implementation + Testing**: Primary agent implements, then `testing` agent adds/updates tests
2. **Implementation + Docs**: Primary agent implements, then `docs-examples` updates documentation
3. **Implementation + Review**: Primary agent implements, then another agent reviews for correctness
4. **Full Pipeline**: Implement → Test → Document → Review

### Model Selection

Use the `model` parameter in the Task tool to select the appropriate model for each agent:

| Model | Use For | Examples |
|-------|---------|----------|
| `opus` | Complex architecture, deep review, critical decisions | MCP server skeleton, Argo client wrapper, cross-checking complex code |
| `sonnet` | Standard implementation tasks (default) | Individual MCP tools, CI setup, documentation, most features |

**Recommended model by task type:**

| Task Type | Primary Agent Model | Review/Cross-check Model |
|-----------|--------------------|-----------------------|
| Core architecture (PIP-10, PIP-13) | `opus` | `opus` |
| MCP tool implementation | `sonnet` | `sonnet` |
| Testing | `sonnet` | - |
| Documentation | `sonnet` | - |
| CI/DevOps | `sonnet` | `sonnet` |
| Code review / Cross-check | - | `opus` |

## Step 6: Execute with Multiple Agents

### Phase 1: Primary Implementation

Delegate to the primary agent with:
- Full issue description
- Any relevant context from existing code
- Clear instruction to implement according to spec
- **Model**: `opus` for core architecture, `sonnet` for standard tasks

### Phase 2: Supporting Agents (run as appropriate)

After primary implementation, invoke supporting agents:

**If code was written, consider:**
- `testing` agent (model: `sonnet`): "Review the implementation of [issue] and add appropriate unit tests"
- `go-developer` agent (model: `sonnet`): "Review the implementation of [issue] for Go best practices and potential issues"

**If a new feature was added, consider:**
- `docs-examples` agent (model: `sonnet`): "Update documentation to reflect the new [feature] implementation"

**If MCP tool was implemented, consider:**
- `testing` agent (model: `sonnet`): "Add unit tests for the new [tool_name] MCP tool"
- `docs-examples` agent (model: `sonnet`): "Add usage example for the new [tool_name] tool to the README"

### Phase 3: Cross-Check (optional but recommended)

For complex implementations, have a different agent verify using `opus` for deeper analysis:
- `go-developer` (model: `opus`): "Review this implementation for correctness, error handling, and edge cases"
- `kubernetes-argo` (model: `opus`): "Verify the Argo client usage is correct and follows best practices"

### Phase 4: Create Follow-up Tasks (as needed)

During implementation, agents may identify improvements or issues that are out of scope for the current task. Instead of expanding scope, create new Linear issues for later:

**When to create follow-up tasks:**
- Technical debt that should be addressed later
- Refactoring opportunities discovered during implementation
- Additional test coverage needed
- Documentation improvements
- Performance optimizations
- Edge cases that need handling
- Related features that would be nice to have

**How to create a follow-up task:**
```
mcp__linear-server__create_issue(
  team: "Pipekit",
  project: "mcp-for-argo-workflows",
  title: "Brief description of the task",
  description: "## Context\n\nDiscovered while implementing [PIP-X].\n\n## Problem/Opportunity\n\n[Description]\n\n## Suggested Approach\n\n[How to fix/improve]\n\n## Dependencies\n\n- PIP-X (if applicable)",
  labels: ["technical-debt"] or ["enhancement"] or ["testing"] as appropriate
)
```

**Guidelines for follow-up tasks:**
- Keep the current task focused - don't expand scope
- Be specific about what needs to be done
- Reference the original issue for context
- Use appropriate labels: `technical-debt`, `enhancement`, `testing`, `docs`, `bug`
- Don't create follow-ups for trivial issues - fix them now if quick

## Step 7: Verify Implementation

1. **Run linter** - `make lint` (fix any issues)
2. **Run tests** - `make test` (ensure passing)
3. **Manual check** - Verify against acceptance criteria from issue

## Step 8: Commit and Create Pull Request

### Commit Changes

Create commits on the feature branch with message format:
```
[PIP-X] Brief description of implementation

- Detail 1
- Detail 2

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Push and Create PR

```bash
git push -u origin <branch-name>
```

Create a pull request using `gh`:
```bash
gh pr create --title "[PIP-X] Title from Linear issue" --body "## Summary

<Brief description of changes>

## Changes

- Change 1
- Change 2

## Testing

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Manual verification against acceptance criteria

## Linear Issue

Closes PIP-X

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
```

### Link PR to Linear

Add a comment to the Linear issue with the PR link:
```
mcp__linear-server__create_comment(issueId: "<issue-id>", body: "PR created: <pr-url>\n\nAgents involved:\n- [primary agent]: [what they did]\n- [supporting agent]: [what they did]\n\nFollow-up tasks created:\n- PIP-XX: [title] (if any)")
```

## Step 9: Address Code Review Feedback

CI must pass and any reviewer feedback must be addressed before merging.

### Review Process

1. **Wait for CI** - All checks must pass
2. **Address all feedback** - Fix issues raised by reviewers
3. **Commit and push fixes** - Push additional commits to the PR branch

### Addressing Review Comments

For each review comment:

1. **Read the feedback carefully** - Understand what issue was identified
2. **Make the fix** - Update the code to address the concern
3. **Commit with context**:
   ```
   Address review feedback: <brief description>

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```
4. **Push the fix**: `git push`

### If You Disagree with Feedback

If a suggestion isn't applicable:
1. Reply to the comment explaining why
2. Document the reasoning in the code if needed

## Step 10: Merge PR and Update Linear

Once CI is green and any review feedback is addressed:

### Merge the PR

```bash
gh pr merge --squash --delete-branch
```

Or merge via GitHub UI with squash merge.

### Update Linear

1. Move issue to "Done":
   ```
   mcp__linear-server__update_issue(id: "<issue-id>", state: "Done")
   ```

2. Add completion comment:
   ```
   mcp__linear-server__create_comment(issueId: "<issue-id>", body: "PR merged. Implementation complete.")
   ```

### Clean Up Local Branch

```bash
git checkout main
git pull origin main
git branch -d <branch-name>
```

## Error Handling

If implementation fails:
1. Keep issue in "In Progress"
2. Add comment describing the blocker
3. Report to user with details
4. Suggest resolution steps

## Examples

### Example 1: MCP Tool Implementation (PIP-15)

1. **Primary**: `mcp-tool-implementer` (model: `sonnet`) - Implements submit_workflow tool
2. **Supporting**: `testing` (model: `sonnet`) - Adds unit tests for the tool handler
3. **Supporting**: `docs-examples` (model: `sonnet`) - Updates README with tool description

### Example 2: Core Architecture (PIP-10)

1. **Primary**: `go-developer` (model: `opus`) - Implements MCP server skeleton
2. **Supporting**: `testing` (model: `sonnet`) - Adds basic server tests
3. **Cross-check**: `mcp-tool-implementer` (model: `opus`) - Verifies tool registration pattern is correct

### Example 3: CI Setup (PIP-8)

1. **Primary**: `ci-devops` (model: `sonnet`) - Creates GitHub Actions workflow
2. **Cross-check**: `go-developer` (model: `sonnet`) - Verifies Go-specific CI configuration

---

Begin by fetching the issue details for: $ARGUMENTS
