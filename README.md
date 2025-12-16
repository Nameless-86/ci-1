# CI-1 Repository - Workflow Explanation

## Overview

This repository (`ci-1`) contains code that, when its dependencies change (specifically when `go.sum` changes), automatically triggers a build process in a separate repository (`ci-2`). The `ci-2` repository depends on `ci-1` as a Go module, so when `ci-1` changes, `ci-2` needs to update its dependency and rebuild.

## The Big Picture

Here's how the two repositories work together:

1. **Developer makes changes to `ci-1`** - This could be adding new dependencies, updating existing ones, or modifying code
2. **When `go.sum` changes in `ci-1`** - This file tracks the exact checksums of all dependencies
3. **CI-1 workflow triggers** - A GitHub Actions workflow detects the `go.sum` change
4. **Repository dispatch event sent** - The workflow sends a special event to the `ci-2` repository
5. **CI-2 workflow receives event** - The `ci-2` repository has a workflow listening for this event
6. **CI-2 updates and builds** - The `ci-2` workflow updates its dependency on `ci-1`, builds a binary, and commits it back

## What is repository_dispatch?

`repository_dispatch` is a GitHub feature that allows one repository to send a custom event to another repository. Think of it like a message system between repositories:

- **Sender (ci-1)**: Uses the GitHub API to send a custom event with a specific `event_type`
- **Receiver (ci-2)**: Has a workflow that listens for that specific `event_type` and runs when it receives it

This is useful when you want one repository's changes to trigger actions in another repository, without them being directly connected in code.

## Workflow File: `.github/workflows/dispatch-ci2-on-go-sum.yaml`

Let's break down this file line by line:

### Workflow Name
```yaml
name: Repository Dispatch Ci-2 on Go Sum changes
```
- This is just a human-readable name for the workflow
- Shows up in the GitHub Actions UI

### Trigger Conditions
```yaml
on:
    workflow_dispatch:
    push:
        branches: [main]
        paths:
            - "go.sum"
    pull_request:
        branches: [main]
        paths:
            - "go.sum"
```

This section defines **when** the workflow runs:

- **`workflow_dispatch:`** - Allows manual triggering from the GitHub Actions UI (useful for testing)
- **`push:`** - Triggers when code is pushed to a branch
  - **`branches: [main]`** - Only triggers on pushes to the `main` branch
  - **`paths: ["go.sum"]`** - Only triggers if the `go.sum` file was changed in that push
- **`pull_request:`** - Similar to push, but for pull requests targeting `main`

**Why `paths` filter?** We only want to trigger `ci-2` when dependencies actually change, not for every code change. The `go.sum` file changes when dependencies are added, removed, or updated.

### Permissions
```yaml
permissions:
    contents: read
```
- Defines what permissions this workflow needs
- `contents: read` - Only needs to read repository contents (checkout code)
- This is a security best practice - only grant minimum required permissions

### Jobs Section
```yaml
jobs:
    dispatch-ci2:
        runs-on: ubuntu-latest
```

- **`jobs:`** - A workflow can have multiple jobs that run in parallel or sequence
- **`dispatch-ci2:`** - The name of this specific job (can be anything)
- **`runs-on: ubuntu-latest`** - Runs on a fresh Ubuntu Linux virtual machine

### Steps - Checkout
```yaml
        steps:
            - uses: actions/checkout@v4
```

- **`steps:`** - A job consists of multiple steps that run sequentially
- **`uses: actions/checkout@v4`** - This is a pre-built action that checks out (downloads) your repository code
- The `@v4` means it uses version 4 of the checkout action
- This makes your code available in the workflow's file system

### Steps - Trigger CI-2
```yaml
            - name: trigger ci-2 build
              env:
                  CI2_PAT: ${{ secrets.CI2_PAT }}
              run: |
                  if [ -z "${CI2_PAT}" ]; then
                    echo "missing access token"; exit 1
                  fi

                  set -e
                  echo "calling repository dispatch..."
                  curl -i --fail \
                    -H "Accept: application/vnd.github+json" \
                    -H "Authorization: Bearer ${CI2_PAT}" \
                    https://api.github.com/repos/Nameless-86/ci-2/dispatches \
                    -d '{"event_type": "go_sum_changed-at-ci-1", "client_payload":{}}'
```

Let's break this down:

- **`name:`** - A descriptive name for this step (shows in logs)
- **`env:`** - Sets environment variables for this step
  - **`CI2_PAT: ${{ secrets.CI2_PAT }}`** - Loads a secret called `CI2_PAT` from repository secrets
    - `${{ }}` is GitHub Actions syntax for accessing variables/secrets
    - A PAT (Personal Access Token) is needed to authenticate API calls to another repository

- **`run: |`** - Executes shell commands (the `|` allows multi-line commands)

The shell script does:

1. **`if [ -z "${CI2_PAT}" ]; then ...`** - Checks if the PAT secret exists
   - `-z` means "is empty"
   - If empty, prints error and exits with code 1 (failure)

2. **`set -e`** - Makes the script exit immediately if any command fails

3. **`echo "calling repository dispatch..."`** - Prints a log message

4. **`curl`** - Makes an HTTP request to GitHub's API:
   - **`-X POST`** (implied by `-d`) - HTTP POST method
   - **`-i`** - Include response headers in output (for debugging)
   - **`--fail`** - Treat HTTP errors as failures
   - **`-H "Accept: application/vnd.github+json"`** - Tells GitHub we want JSON response
   - **`-H "Authorization: Bearer ${CI2_PAT}"`** - Authenticates using the PAT
   - **`https://api.github.com/repos/Nameless-86/ci-2/dispatches`** - The API endpoint
     - `/repos/{owner}/{repo}/dispatches` - Sends a repository_dispatch event
   - **`-d '{"event_type": "go_sum_changed-at-ci-1", "client_payload":{}}'`** - The request body
     - `event_type` - The type of event (must match what `ci-2` is listening for)
     - `client_payload` - Optional data to send (empty in our case)

## Required Secrets

You need to set up this secret in `ci-1` repository settings:

- **`CI2_PAT`** - A Personal Access Token with `repo` scope that has access to the `ci-2` repository

To create a PAT:
1. GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token with `repo` scope
3. Copy the token and add it as a secret named `CI2_PAT` in `ci-1` repository settings

## Workflow Execution Flow

1. Developer pushes changes to `main` branch that modify `go.sum`
2. GitHub detects the push matches the workflow trigger conditions
3. Workflow starts on a fresh Ubuntu runner
4. Step 1: Checks out `ci-1` code
5. Step 2: Validates PAT exists, then calls GitHub API to send repository_dispatch event to `ci-2`
6. GitHub receives the API call and creates a `repository_dispatch` event in `ci-2`
7. The `ci-2` workflow (if configured correctly) picks up this event and starts running

## Testing the Workflow

You can manually trigger this workflow:
1. Go to `ci-1` repository → Actions tab
2. Click on "Repository Dispatch Ci-2 on Go Sum changes"
3. Click "Run workflow" button
4. This will trigger the workflow without needing to push code

This is useful for testing if the repository_dispatch is working correctly.
