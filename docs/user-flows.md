# TickCats User Flow Diagrams

These Mermaid diagrams summarize the v1 flows from the PRD and current CLI implementation. Workflow state is derived from ticket column folder location under `.tickcats/`.

## Initialize local board

```mermaid
flowchart TD
    A["User runs tickcats init"] --> B["Create .tickcats/"]
    B --> C["Create default columns\nbacklog/ ready/ doing/ done/ wont-do/"]
    C --> D["Ensure .gitignore contains .tickcats/"]
    D --> E["Repo-local private board is ready"]
```

## Create a ticket

```mermaid
flowchart TD
    A[User creates ticket] --> B{Kind command?}
    B -->|new feat| C[Use Feat: prefix]
    B -->|new task| D[Use Task: prefix]
    B -->|new bug| E[Use Bug: prefix]
    C --> F["Build markdown with frontmatter"]
    D --> F
    E --> F
    F --> G["Set title, priority, created, updated"]
    G --> H["Include Acceptance Criteria section"]
    H --> I["Write ticket markdown file"]
    I --> J["Store in .tickcats/backlog/"]
```

## Board navigation and workflow movement

```mermaid
flowchart LR
    A["Open TickCats board"] --> B["Load configured .tickcats folders"]
    B --> C["Show Backlog column"]
    B --> D["Show Ready column"]
    B --> E["Show Doing column"]
    B --> F["Show Done column"]
    B --> W["Show Won't Do column"]
    C -. h/l .-> D
    D -. h/l .-> E
    E -. h/l .-> F
    F -. h/l .-> W
    C -. j/k .-> C
    D -. j/k .-> D
    E -. j/k .-> E
    F -. j/k .-> F
    W -. j/k .-> W
    G["Move selected ticket"] --> H{"Target column"}
    H -->|Backlog| C
    H -->|Ready| D
    H -->|Doing| E
    H -->|Done| F
    H -->|Won't Do| W
```

## Pick next ready ticket

```mermaid
flowchart TD
    A["User invokes pick-next"] --> B["Load board from .tickcats/"]
    B --> C["Read only ready/ tickets"]
    C --> D{"Ticket has title?"}
    D -->|No| X["Exclude"]
    D -->|Yes| E{"Acceptance Criteria non-empty?"}
    E -->|No| X
    E -->|Yes| F{"Title has [blocked]?"}
    F -->|Yes| X
    F -->|No| G{"Title has [to refine]?"}
    G -->|Yes| X
    G -->|No| H["Eligible candidate"]
    H --> I["Sort by priority P0 > P1 > P2 > P3"]
    I --> J["Break priority ties by oldest created"]
    J --> K{"Same priority and created?"}
    K -->|Yes| L["Show tied candidates for manual selection"]
    K -->|No| M["Recommend single next ticket"]
    X --> N{"Any candidates left?"}
    N -->|No| O["No ready ticket found"]
    N -->|Yes| I
```

## Inspect and refine a ticket

```mermaid
flowchart TD
    A["User selects ticket"] --> B["Open detail view"]
    B --> C["Read full markdown content"]
    C --> D["Show metadata and body"]
    D --> E{"Action"}
    E -->|Scroll| F["j/k navigate content"]
    E -->|Edit metadata| G["Update frontmatter"]
    E -->|Open editor| H["Launch external editor"]
    G --> I["Save markdown ticket"]
    H --> I
    I --> J["Reload board and pick-next status"]
```

## Command palette actions

```mermaid
flowchart TD
    A["User opens command palette"] --> B{Command}
    B -->|New Feature| C["Create Feat ticket"]
    B -->|New Task| D["Create Task ticket"]
    B -->|New Bug| E["Create Bug ticket"]
    B -->|Move to column| F["Move markdown file between column folders"]
    B -->|Edit Metadata| G["Update selected ticket frontmatter"]
    B -->|Open in Editor| H["Open selected markdown file externally"]
    B -->|Pick Next| I["Run pick-next rule"]
```
