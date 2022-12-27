---
name: Bug report
about: Create a report to help us improve
title: ''
labels: bug
assignees: ''

---

**What happened**:

**Environment**:

- OS (e.g. `cat /etc/os-release`):
- Kernel (e.g. `uname -a`):
- Service Version (Execute: `kubectl exec -n onecloud $(kubectl get pods -n onecloud | grep climc | awk '{print $1}') -- climc version-list`):
<!--
- Version (e.g. `climc version-list`):
-->
