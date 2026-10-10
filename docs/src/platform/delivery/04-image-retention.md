---
title: Image retention
---

Every build pushes a new image tag to Docker Hub, and Docker Hub has no retention rules of its own.
The `registry-cleaner` job deletes old tags on a schedule, following a policy kept in Git, much like
<a href="https://docs.aws.amazon.com/AmazonECR/latest/userguide/LifecyclePolicies.html" target="_blank" rel="noopener">ECR
lifecycle policies</a>.

## How a policy decides

A policy lists repository patterns and a set of **prioritized rules**.

**Which policy applies.** The job lists every repository in the namespace and gives each one the
policy with the **most specific** matching pattern. In a pattern, `*` matches any characters,
including `/`. An exact name beats any pattern, and between patterns the one with more literal
characters wins, so `portfolio` beats `product/*`, which beats `*`. The winning policy applies
alone: its rules replace those of broader patterns rather than adding to them. Two equally specific
patterns matching the same repository fail that repository instead of picking one at random. A
repository no pattern matches is left untouched.

**Which tags a policy deletes.** Rules run by ascending `priority`, and the **first rule that
matches a tag decides** whether it is kept or deleted; lower priorities never see that tag. A rule
has:

| Field         | Meaning                                                                                |
| ------------- | -------------------------------------------------------------------------------------- |
| `priority`    | Unique within the policy; lower runs first                                             |
| `action`      | `keep` or `delete`                                                                     |
| `tagPattern`  | Optional regular expression selecting the tags the rule applies to (all tags if unset) |
| `newest`      | Optional: matches the N most recently pushed tags among those the pattern selects      |
| `olderThan`   | Optional: matches tags pushed longer ago than this (Go durations, plus `d` for days)   |
| `description` | Optional, shown in the logs next to each decision                                      |

A rule sets at most one of `newest` and `olderThan`; with neither, it matches every tag it selects.
A tag no rule matches is **kept**, so a gap in the rules can never delete anything.

**Why explicit actions, not ECR's model.** In ECR every rule expires images and a tag claimed by one
rule is out of reach of the next, so "keep the 10 newest, then delete anything older than 5 days"
can't be written. With a `keep` rule at a higher priority than a `delete` rule, it is two lines, and
the order in which they apply is written down rather than implied.

The job doesn't look at what is deployed: **the policy alone decides what survives**. Size the
`keep` rules so every tag still running is covered, for example a `newest` count larger than the
number of tags deployed at once. A tag with no push date is always kept, whatever the rules say,
since its age can't be checked.

## Configuration

The policy lives under `config` in the chart's
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/registry-cleaner/chart/values.yaml" target="_blank" rel="noopener"><code>values.yaml</code></a>
and is mounted into the job as a ConfigMap. It starts with a single `*` policy; give a repository
its own policy to override it. A pattern may appear in only one policy. Unknown keys are rejected,
so a typo such as `olderthan` fails the run instead of silently turning a rule into a catch-all.

`dryRun: true` logs every tag the run would delete and deletes nothing.

## Why a CronJob

The work is a short batch with a start and an end, so a
<a href="https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/" target="_blank" rel="noopener">CronJob</a>
fits better than a long-running Deployment: nothing runs between runs, `concurrencyPolicy: Forbid`
stops two runs from deleting at the same time, and a failed run shows up as a failed Job. Cleaning
up right after each push wouldn't work: tags get older without any new push, so a schedule is needed
anyway.

## Gotchas

- **One failing repository doesn't stop the others.** The run carries on and fails at the end,
  listing every repository that failed.
- **A rescheduled pod can lose its image.** A running pod keeps working after its tag is deleted,
  but fails with `ImagePullBackOff` the next time its image has to be pulled. Widen the policy
  rather than restoring by hand.
- **Rate limits.** Docker Hub throttles its API, and a first run on a large backlog can hit the
  limit. The client waits for the `Retry-After` delay and retries.
- **Token permissions.** Deleting tags needs a Docker Hub token with delete rights. If deletes
  return `403`, use an organization access token with the Tag Admin scope.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/registry-cleaner" target="_blank" rel="noopener"><code>platform/services/registry-cleaner/</code></a>
  — the Go job, its Dockerfile and its Nx project
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/registry-cleaner/internal/retention/evaluate.go" target="_blank" rel="noopener"><code>internal/retention/evaluate.go</code></a>
  — how a tag is kept or deleted
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/registry-cleaner/chart/values.yaml" target="_blank" rel="noopener"><code>chart/values.yaml</code></a>
  — schedule and retention policy
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/registry-cleaner/chart/templates/secret.yaml" target="_blank" rel="noopener"><code>chart/templates/secret.yaml</code></a>
  — where the Docker Hub credentials come from
