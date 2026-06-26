---
name: feedback-cli-cobra
description: Use spf13/cobra for the CLI (Phase 7)
metadata:
  type: feedback
---

Use spf13/cobra for implementing the CLI in Phase 7 (cmd/cli/).

**Why:** User explicitly requested it.

**How to apply:** When implementing cmd/cli/, install cobra and structure all commands (run, logs, status, agent start, trigger) as cobra.Command instances with a root command.
