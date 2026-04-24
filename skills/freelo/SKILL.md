---
name: freelo
description: >
  Use when interacting with Freelo.io project management — managing projects, tasklists,
  tasks, subtasks, comments, time tracking, work reports, labels, notes, files, custom fields,
  invoices, templates, or searching in Freelo. Trigger this skill whenever the user mentions
  Freelo, wants to manage tasks or projects, track time, create work reports, or do anything
  related to Freelo.io.
---

# Freelo CLI — Agent Skill

You have access to the `freelo` command-line tool for managing Freelo.io projects and tasks.
**Always use the `--agent` flag** for clean, parseable JSON output.

## Authentication
```bash
freelo auth status                        # Check if authenticated
freelo auth login                         # Interactive login
# Or set env vars: FREELO_EMAIL, FREELO_API_KEY
```

## Projects
```bash
freelo projects list                      # List all active projects
freelo projects show <id>                 # Project detail with tasklists
freelo projects create --name "Name" --currency CZK
freelo projects archive <id>
freelo projects activate <id>
freelo projects delete <id>
```

## Tasks
```bash
freelo tasks list                         # List all tasks
freelo tasks list --project <id>          # Tasks in a project
freelo tasks list --search "keyword"      # Search tasks
freelo tasks list --worker <user-id>      # Tasks assigned to user
freelo tasks show <id>                    # Task detail
freelo tasks create --project <id> --tasklist <id> --name "Name"
freelo tasks create --project <id> --tasklist <id> --name "Name" --due-date 2025-12-31 --priority h --worker <uid>
freelo tasks edit <id> --name "New" --priority m --due-date 2025-12-31 --worker <uid>
freelo tasks finish <id>                  # Complete task
freelo tasks activate <id>               # Reopen task
freelo tasks move <id> --tasklist <tid>   # Move to another tasklist
freelo tasks description <id>            # Get description
freelo tasks description <id> --set "<p>HTML</p>"  # Set description
```

## Subtasks
```bash
freelo subtasks list --task <id>          # List subtasks
freelo subtasks create --task <id> --name "Name" --due-date 2025-12-31 --worker <uid> --priority h
freelo subtasks show <id>                 # Subtask detail
freelo subtasks finish <id>               # Complete subtask
freelo subtasks activate <id>             # Reopen subtask
freelo subtasks delete <id>
```

## Tasklists
```bash
freelo tasklists list --project <id>      # List tasklists in project
freelo tasklists show <id>                # Detail with tasks
freelo tasklists create --project <id> --name "Sprint 1"
```

## Search
```bash
freelo search "keyword"                   # Search everything
freelo search "bug" --type task           # Types: task, subtask, project, tasklist, file, comment
freelo search "design" --project <id>     # Search within project
```

## Comments
```bash
freelo comments list --project <id>
freelo comments create --task <id> --content "Text"
freelo comments edit <id> --content "Updated"
freelo comments delete <id>
```

## Labels
```bash
freelo labels list                        # All available labels
freelo labels create --name "Bug" --color "#ff0000"
freelo labels add-to-task --task <id> --name "Bug"
freelo labels add-to-task --task <id> --uuid <label-uuid>
freelo labels remove-from-task --task <id> --uuid <label-uuid>
freelo labels add-to-project --project <id> --name "Priority" --color "#00ff00"
freelo labels remove-from-project --project <id> --label-id <id>
freelo labels edit <label-id> --name "New name" --color "#0000ff"
freelo labels delete <label-id>
```

## Time Tracking
```bash
freelo tracking start --task <id>         # Start timer
freelo tracking start --task <id> --note "Working on feature"
freelo tracking status                    # Check active timer
freelo tracking stop                      # Stop timer → creates work report
```

## Work Reports
```bash
freelo reports list --project <id> --user <uid>
freelo reports create --task <id> --minutes 60 --date 2025-01-15 --note "Feature work"
freelo reports edit <id> --minutes 90 --note "Updated"
freelo reports delete <id>
```

## Notes
```bash
freelo notes create --project <id> --name "Title" --content "<p>HTML content</p>"
freelo notes show <id>
freelo notes edit <id> --name "New title" --content "<p>Updated</p>"
freelo notes delete <id>
```

## Files
```bash
freelo files list --project <id>          # List files and docs
freelo files list --type file             # Types: directory, link, file, document
freelo files download <file-uuid> --output ./local-file.pdf
freelo files upload ./document.pdf        # Upload (max 100MB)
```

## Custom Fields
```bash
freelo custom-fields types                # List available types (text, number, enum)
freelo custom-fields list --project <id>  # List fields in project
freelo custom-fields create --project <id> --name "Estimate" --type-uuid <uuid>
freelo custom-fields rename <field-uuid> --name "New Name"
freelo custom-fields delete <field-uuid>
freelo custom-fields restore <field-uuid>
freelo custom-fields set-value --task <id> --field-uuid <uuid> --value "42"
freelo custom-fields delete-value <value-uuid>
# Enum fields:
freelo custom-fields enum-options <field-uuid>
freelo custom-fields enum-create <field-uuid> --name "Option A" --color "#ff0000"
freelo custom-fields enum-edit <enum-uuid> --name "Option B"
freelo custom-fields enum-delete <enum-uuid> --force
freelo custom-fields set-enum-value --task <id> --field-uuid <uuid> --enum-uuid <uuid>
```

## Templates
```bash
freelo templates list                     # List template projects
freelo templates create-project --template <id> --name "New Project"
freelo templates create-tasklist --template <id>
freelo templates create-task --template <id>
```

## Pinned Items
```bash
freelo pinned list --project <id>
freelo pinned create --project <id> --link "https://..." --title "Design doc"
freelo pinned delete <id>
```

## Users & Workers
```bash
freelo users me                           # Current user info
freelo users list                         # All coworkers
freelo workers list --project <id>        # Workers in a project
freelo workers invite --emails "a@b.com,c@d.com" --projects "123,456"
freelo workers remove --project <id> --emails "a@b.com"
```

## Out of Office
```bash
freelo out-of-office status --user <id>
freelo out-of-office enable --user <id> --from "2025-07-01 00:00:00" --to "2025-07-14 23:59:59"
freelo out-of-office disable --user <id>
```

## Notifications
```bash
freelo notifications list                 # All notifications
freelo notifications list --unread        # Unread only
freelo notifications read <id>
freelo notifications unread <id>
```

## Invoices
```bash
freelo invoices list --project <id>
freelo invoices show <id>
freelo invoices mark-invoiced <id> --url "https://..." --subject "Invoice #123"
```

## Events (Activity Log)
```bash
freelo events list --project <id> --user <uid>
```

## Raw API Access
```bash
freelo api get /projects
freelo api post /search --data '{"query":"test"}'
freelo api put /task/123 --data '{"name":"Updated"}'
freelo api delete /task/123
```

## Environments
```bash
freelo projects list              # Production (default)
freelo projects list --dev        # Development (requires FREELO_DEV_URL env var)
freelo auth login --dev           # Login to dev (separate credentials)
```

## Output Formats
- `--agent` — Raw JSON (best for AI agents, always use this)
- `--json` / `-j` — JSON with envelope `{ok, data, summary, breadcrumbs}`
- `--quiet` / `-q` — Minimal text
- `--ids-only` — Only IDs, one per line
- `--count` — Only count

## Key Conventions
- **IDs**: integers (tasks, projects, users) or UUIDs (custom fields, labels)
- **Priorities**: `h` (high), `m` (medium), `l` (low), null (none)
- **Dates**: YYYY-MM-DD format
- **Pagination**: `--page <n>` (0-indexed)
- **Rate limit**: 25 req/min (handled automatically by CLI)
- **Currency**: amounts = value × 100, string format (e.g. "100025" = 1000.25 CZK)
