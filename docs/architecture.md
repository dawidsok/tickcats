# TickCats Architecture Diagrams

These Mermaid diagrams describe the v1 local CLI/TUI architecture. The source of truth is the filesystem: ticket status comes from the column folder containing each markdown file, not from ticket frontmatter.

## System context

```mermaid
flowchart LR
    U[Keyboard-first solo developer] -->|commands / hotkeys| A[TickCats CLI/TUI]
    A -->|read/write markdown| S[(Repo-local .tickcats/)]
    A -->|open selected ticket| E[External editor]
    A -->|ensure ignored| G[.gitignore]

    subgraph Repo[User repository]
        S
        G
    end
```

## Local storage layout

```mermaid
flowchart TD
    R[User repository] --> T[".tickcats/"]
    T --> B["backlog/"]
    T --> Y["ready/"]
    T --> D["doing/"]
    T --> N["done/"]
    T --> W["wont-do/"]
    T --> X["custom-column/"]
    B --> B1["ticket markdown files"]
    Y --> Y1["ticket markdown files"]
    D --> D1["ticket markdown files"]
    N --> N1["ticket markdown files"]
    W --> W1["ticket markdown files"]
    X --> X1["ticket markdown files"]
    T --> CFG["config.json\ncolumn order/display names"]

    M[Ticket markdown] --> F[Frontmatter: title, priority, created, updated]
    M --> C[Body: Context, Acceptance Criteria]
```

## Package-level architecture

```mermaid
flowchart TD
    CLI[cmd/tickcats]
    Store[internal/store]
    Ticket[internal/ticket]
    FS[(Filesystem)]

    CLI -->|init, new, list, move, pick-next| Store
    CLI -->|create markdown, parse kind/title helpers| Ticket
    Store -->|load, move, initialize/sync column folders| FS
    Store -->|parse and validate ticket files| Ticket
    Ticket -->|markdown/frontmatter parsing| TicketParser[Ticket parser]
    Ticket -->|priority ordering| Priority[Priority rules]
    Ticket -->|labels and kind prefixes| Title[Title parser]
```

## Runtime command flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/tickcats
    participant Store as internal/store
    participant Ticket as internal/ticket
    participant FS as .tickcats filesystem

    User->>CLI: tickcats <command>
    alt init
        CLI->>Store: Init(".")
        Store->>FS: mkdir backlog/ ready/ doing/ done/ wont-do/
        Store->>FS: append .tickcats/ to .gitignore if missing
    else new feat|task|bug <title>
        CLI->>Store: Init(".")
        CLI->>Ticket: NewMarkdown(kind, title, P2, now)
        CLI->>FS: write .tickcats/backlog/<slug>.md
    else list
        CLI->>Store: LoadBoard(".")
        Store->>FS: read configured column folders and .md files
        Store->>Ticket: ParseMarkdown(file)
        Store-->>CLI: Board columns + warnings
        CLI-->>User: grouped ticket list
    else move
        CLI->>Store: Move(root, name, from, to)
        Store->>FS: read source ticket
        Store->>Ticket: ParseMarkdown(source)
        Store->>FS: rename into target column folder
    else ids migrate
        CLI->>Store: MigrateIDs(root)
        Store->>FS: add missing frontmatter ids
        Store->>FS: rename migrated files to id-based filenames
    else pick-next
        CLI->>Store: LoadBoard(".")
        Store->>Ticket: ParseMarkdown(each ticket)
        CLI->>Store: PickNext(board)
        Store-->>CLI: pick result or tie candidates
        CLI-->>User: recommendation
    end
```

## Pick-next rule architecture

```mermaid
flowchart TD
    A[Board loaded from filesystem] --> B[Use Ready column only]
    B --> C[Filter with IsReadyForPick]
    C --> D[State == ready]
    C --> E[Non-empty title]
    C --> F[Non-empty Acceptance Criteria]
    C --> G[No [blocked] label]
    C --> H[No [to refine] label]
    D --> I[Eligible candidates]
    E --> I
    F --> I
    G --> I
    H --> I
    I --> J[Sort by Priority.HigherThan]
    J --> K[Sort ties by Created ascending]
    K --> L[Sort exact ties by filename]
    L --> M{Multiple same rank?}
    M -->|Yes| N[Needs manual choice]
    M -->|No| O[Return single recommendation]
```

## Data model

```mermaid
classDiagram
    class Board {
        map~State, []StoredTicket~ Columns
        []Warning Warnings
    }

    class StoredTicket {
        string Path
        string Name
        State State
        Ticket Ticket
    }

    class Ticket {
        string ID
        string Title
        ParsedTitle ParsedTitle
        Priority Priority
        time Created
        time Updated
        optional date Deadline
        string Body
        bool HasAcceptanceCriteria
    }

    class ParsedTitle {
        string Raw
        []string Labels
        Kind Kind
        string Text
        bool HadPrefix
        Blocked() bool
        ToRefine() bool
        NormalizedTitle() string
    }

    class Priority {
        P0
        P1
        P2
        P3
        Rank() int
        HigherThan(other) bool
    }

    Board "1" --> "many" StoredTicket
    StoredTicket --> Ticket
    Ticket --> ParsedTitle
    Ticket --> Priority
```

## Planned v1 TUI boundary

```mermaid
flowchart TD
    TUI[TUI layer]
    Commands[Command palette and hotkeys]
    BoardView[Kanban board view]
    DetailView[Ticket detail view]
    AppService[Application actions]
    Store[internal/store]
    Ticket[internal/ticket]
    Editor[External editor]
    FS[(.tickcats/)]

    TUI --> Commands
    TUI --> BoardView
    TUI --> DetailView
    Commands --> AppService
    BoardView --> AppService
    DetailView --> AppService
    AppService --> Store
    AppService --> Ticket
    AppService --> Editor
    Store --> FS
```
