<p align="center">
  <img src="docs/logo.svg" width="160" alt="kubectl-harvest logo">
</p>

<h1 align="center">kubectl-harvest</h1>

<p align="center"><strong>Delete unused Kubernetes resources.</strong><br>
<em>A kubectl plugin that reaps the leftovers every cluster accumulates.</em></p>

<p align="center">
  <a href="https://github.com/vivektech/kubectl-harvest/actions/workflows/test.yml"><img src="https://github.com/vivektech/kubectl-harvest/actions/workflows/test.yml/badge.svg" alt="Test"></a>
  <a href="https://github.com/vivektech/kubectl-harvest/actions/workflows/lint.yml"><img src="https://github.com/vivektech/kubectl-harvest/actions/workflows/lint.yml/badge.svg" alt="Lint"></a>
  <a href="https://github.com/vivektech/kubectl-harvest/releases"><img src="https://img.shields.io/github/v/release/vivektech/kubectl-harvest?logo=github&label=release" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/vivektech/kubectl-harvest"><img src="https://pkg.go.dev/badge/github.com/vivektech/kubectl-harvest.svg" alt="Go Reference"></a>
</p>

<p align="center">
  <a href="#kubernetes-compatibility"><img src="https://img.shields.io/badge/kubernetes-1.21%2B-326CE5?logo=kubernetes&logoColor=white" alt="Kubernetes 1.21+"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue?logo=apache" alt="License: Apache-2.0"></a>
  <a href="INSTALL.md"><img src="https://img.shields.io/badge/platforms-linux%20%7C%20macOS%20%7C%20windows-informational?logo=linux" alt="Platforms"></a>
</p>

---

`kubectl-harvest` is a kubectl plugin that finds and deletes **unused** Kubernetes resources — the leftovers that pile up in every cluster: completed Jobs, unreferenced ConfigMaps and Secrets, orphaned volumes, scaled-away ReplicaSets.

It is the maintained successor of [micnncim/kubectl-reap](https://github.com/micnncim/kubectl-reap), which was archived and no longer works on modern clusters. Full credits to the original author and contributors — see [Credits](#credits).

## How it works, in simple words

Think of it as a two-step process, exactly like a careful human would do it:

**Step 1 — Make a list of candidates.** You run `kubectl harvest <resource-type>` (e.g. `cm`). The plugin asks the cluster for all objects of that type — using the exact same machinery `kubectl delete` uses, so namespaces, contexts, label selectors, and your kubeconfig all behave identically.

**Step 2 — For each candidate, ask: "is anyone still using this?"** The plugin has a per-resource-type checklist (detailed below). It only deletes a resource when it can prove nothing references it. If the answer is "maybe" — for example a scaled-to-zero Deployment whose ConfigMap would be needed again the moment it scales up — the resource is **kept**.

For example, before deleting a ConfigMap it checks every Pod and every workload template (Deployments, StatefulSets, DaemonSets, Jobs, CronJobs, ReplicaSets) in the namespace for references — mounted volumes, `env` values, `envFrom`, init containers, everything. Only a ConfigMap no one points at gets deleted.

### What "unused" means per resource type

| Resource | Deleted only when... |
|---|---|
| **Pod** | Not in `Running` phase (Completed/Failed/Unknown pods) |
| **ReplicaSet** | Scaled to 0 replicas **and** older than the newest `--keep-revisions` revisions of its Deployment (default keeps 5 for rollback) |
| **ConfigMap** | Not referenced by any Pod or workload template (env, envFrom, volumes, projected volumes, init containers) |
| **Secret** | Not referenced by any Pod or workload template, ServiceAccount (tokens + image pull secrets), Ingress (TLS + nginx auth annotations), or ExternalSecret |
| **PersistentVolume** | Not `Bound`, and matches no PersistentVolumeClaim (capacity, storage class, volume mode, access modes) |
| **PersistentVolumeClaim** | Not referenced by any Pod or workload template |
| **Job** | Completed (`completionTime` is set) |
| **PodDisruptionBudget** | Selects no Pods and no workload templates |
| **HorizontalPodAutoscaler** | Its scale target no longer exists |
| **NetworkPolicy** | Selects no Pods and no workload templates |

## Safeguards — what can never be deleted

This is the part we verified line by line, with unit tests for each rule:

**Hard rules — these resources are never touched, period:**

1. **Anything in the `kube-system` namespace** — always skipped, even with `--all-namespaces`.
2. **Running Pods** — only non-`Running` phases are candidates.
3. **Bound PersistentVolumes** — a volume serving a claim is never a candidate, regardless of any other check.
4. **ServiceAccount token Secrets** (`kubernetes.io/service-account-token`) — these are credentials; manually created ones may be used by external systems that are invisible to the cluster, so they are never deleted.
5. **The `kube-root-ca.crt` ConfigMap** — every namespace has one; Pods consume it implicitly, so reference scanning can't see it. Never deleted.
6. **Deployment revision history** — when reaping ReplicaSets, the live revision is always kept, and by default the newest **5** revisions of each Deployment are kept too, so `kubectl rollout undo` always has somewhere to go. Control it with `--keep-revisions N` (or `--keep-revisions 0` to keep only the live revision).

**Reference checks — a resource is kept when anything points at it:**

- Pods and workload templates are scanned in their **entirety**: regular containers, init containers, `env`, `envFrom`, volumes, projected volumes, image pull secrets.
- Workload templates count even when no Pods are running (suspended CronJobs, scaled-to-zero Deployments/StatefulSets) — this is what makes the check safe for paused workloads.
- Secrets referenced by Ingress TLS, nginx ingress auth annotations (`auth-secret`, `auth-tls-secret`), ServiceAccount `imagePullSecrets`, and ExternalSecrets (both the `external-secrets.io` and `kubernetes-client.io` operators) are detected as used.
- PDBs and NetworkPolicies whose selectors match **any live Pod or any workload template** are kept.
- Lookups are namespace-scoped, so a ConfigMap named `foo` in namespace A being in use does **not** accidentally protect a different `foo` in namespace B — and does not get namespace B's copy deleted either.

**Process rails:**

- `--dry-run=client` (or `=server`) prints exactly what would be deleted without deleting anything. **Always start here.**
- `--interactive` asks yes/no for every single resource before deleting.
- `--skip-controller-owned` skips Pods managed by controllers (they'd just be recreated).
- Deletions go through the standard Kubernetes API with normal grace periods; RBAC applies as usual.

### Honest limits — please read

No static analysis can be perfect, because **operators and external systems can reference objects by name in ways that are invisible to the API**. Known blind spots:

- A Secret or ConfigMap consumed only by a **CRD-based operator** or an external system (e.g. cert-manager Certificate targets, a CI system pulling a registry Secret) may look unused. The most common cases (Ingress TLS, nginx auth, ExternalSecret operators, SA tokens) are handled explicitly — the rest is why dry-run exists.
- A Pending Pod may be deleted while waiting for resources; use `--skip-controller-owned` for managed Pods.
- Resources needed only in the future (a ConfigMap for a workload you haven't applied yet) look unused by definition.

**Recommendation:** run `kubectl harvest <type> --dry-run=client` first, review the list, then run for real (optionally with `--interactive`).

## Kubernetes compatibility

Built against **Kubernetes 1.36** client libraries (`k8s.io/* v0.36.4`) — no legacy dependencies.

| Kubernetes cluster version | Status | Notes |
| --- | :---: | --- |
| 1.37 and newer | ✅ | Only stable APIs are used, so future minors keep working |
| **1.34 – 1.36** | ✅ | **Tested target** — primary development and CI focus |
| 1.21 – 1.33 | ✅ | Full feature set; every API used (`policy/v1` PDB, `batch/v1` CronJobs, `networking.k8s.io/v1` Ingresses and NetworkPolicies) has been stable since 1.21 |
| 1.19 – 1.20 | ⚠️ | Runs with reduced detection: CronJob and ExternalSecret references are skipped (`batch/v1` CronJobs not yet served); everything else works |
| 1.18 and older | ⚠️ | Additionally no Ingress TLS detection (`networking.k8s.io/v1` not yet served); PDB reaping still works — `policy/v1beta1` and `policy/v1` fields are identical |

✅ = fully supported · ⚠️ = runs safely with some reference detection disabled

The original kubectl-reap was built against Kubernetes 1.19 and used `policy/v1beta1`, which was removed in Kubernetes 1.25 — that is why it stopped working; this project uses `policy/v1` and current libraries.

## Installation

See [INSTALL.md](INSTALL.md) for step-by-step instructions for **macOS (Intel and Apple Silicon), Linux (x86-64 and ARM64), and Windows (x64 and ARM64)** via Homebrew, Krew, prebuilt binaries, and `go install`.

Quick start:

```
brew tap vivektech/kubectl-harvest https://github.com/vivektech/kubectl-harvest
brew install kubectl-harvest
```

or

```
kubectl krew index add vivektech https://github.com/vivektech/kubectl-harvest
kubectl krew install vivektech/harvest
```

or

```
go install github.com/vivektech/kubectl-harvest/cmd/kubectl-harvest@latest
```

Then verify: `kubectl harvest --version`

## Examples

```console
# See what would be deleted, without deleting anything (always start here)
$ kubectl harvest cm --dry-run=client
configmap/config-2 deleted (dry run)
configmap/config-3 deleted (dry run)

# Delete unused ConfigMaps in the current namespace
$ kubectl harvest cm
configmap/config-2 deleted
configmap/config-3 deleted

# Unused ConfigMaps and Secrets in a specific namespace/context
$ kubectl harvest cm,secret -n my-namespace --context my-context

# Across all namespaces (kube-system is always skipped)
$ kubectl harvest po -A

# Ask before each deletion
$ kubectl harvest secret --interactive

# Delete unused ReplicaSets but always keep the newest 10 revisions of each
# Deployment available for rollback
$ kubectl harvest rs --keep-revisions 10

# Delete completed Jobs
$ kubectl harvest jobs
```

## Credits

`kubectl-harvest` is a fork of [micnncim/kubectl-reap](https://github.com/micnncim/kubectl-reap) by [micnncim](https://github.com/micnncim), plus its [contributors](https://github.com/micnncim/kubectl-reap/graphs/contributors). The original idea, design, and the majority of the code are theirs, published under the Apache License 2.0. This project continues their work with modern Kubernetes support, Apple Silicon builds, and the additional features and safeguards described above. The original project is archived; all credit for creating it belongs to its author.

## License

Apache License 2.0 — see [LICENSE](LICENSE). Original code copyright 2020 micnncim; modifications copyright 2026 the kubectl-harvest authors.
