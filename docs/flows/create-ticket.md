# Create Ticket

Form-based ticket creation with optional external editor launch on completion.

## User flow

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TD
    Board["ViewBoard"] -->|"n"| EnterCreate["enterCreate()\ninitialize 4-field form\nKind=Feature, Priority=P2\nToRefine=true, field=0"]
    EnterCreate --> Form["ViewCreate\nform rendered"]

    Form -->|"tab / shift+tab"| CycleField["cycle fields 0→1→2→3→0"]
    Form -->|"h / l (field 0)"| CycleKind["cycle Kind\nFeature / Task / Bug"]
    Form -->|"h / l (field 2)"| CyclePriority["cycle Priority\nP0 / P1 / P2 / P3"]
    Form -->|"space (field 3)"| ToggleRefine["toggle ToRefine checkbox"]
    Form -->|"esc"| Board

    Form -->|"enter\n(field 1 active)"| Validate{"title\nnon-empty?"}
    Validate -->|No| Error["show 'title required' error"]
    Error --> Form
    Validate -->|Yes| Submit["submitCreate()\nstore.Create(kind, title, labels, priority)"]

    Submit --> WriteFile["ticket.GenerateID()\nticket.NewMarkdownFullWithID()\nos.WriteFile → backlog/"]
    WriteFile --> Reload["reloadBoard()"]
    Reload --> SkipCheck{"Config.SkipEditorPrompt?"}

    SkipCheck -->|true| Notify["success notification\nreturn to ViewBoard"]
    Notify --> Board

    SkipCheck -->|false| PostCreate["InteractionPostCreate dialog\n'Open in editor? y/n/d'"]
    PostCreate -->|"n / esc"| Board
    PostCreate -->|"d"| SaveSkip["saveConfig(SkipEditorPrompt=true)\ndismiss dialog"]
    SaveSkip --> Board
    PostCreate -->|"y"| OpenEditor["tea.ExecProcess\nlaunch editor on new file"]
    OpenEditor --> EditorDone["handleEditorFinished()\nreloadBoard()"]
    EditorDone --> Board
```

## Module architecture

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
graph LR
    subgraph TUI
        Update["update.go"]
        Create["create.go\nenterCreate\nsubmitCreate\nupdateCreate\nrenderCreate"]
        PostDlg["post_create_dialog.go\nrenderPostCreateDialog"]
        EditorPkg["editor.go\neditSelected\nhandleEditorFinished"]
        Config["config_view.go\nsaveConfig"]
    end

    subgraph Store
        StoreCreate["create.go\nCreate()"]
        StoreConfig["config.go\nSaveConfig()"]
        Board["board.go\nLoadBoard()"]
    end

    subgraph Ticket
        IDGen["id.go\nGenerateID()"]
        MdGen["markdown.go\nNewMarkdownFullWithID()"]
        TitlePkg["title.go\nNormalizedTitle()"]
    end

    FS[("backlog/\nconfig.json")]

    Update --> Create
    Update --> PostDlg
    Update --> EditorPkg
    Create --> StoreCreate
    Create --> Board
    StoreCreate --> IDGen
    StoreCreate --> MdGen
    StoreCreate --> TitlePkg
    StoreCreate --> FS
    Config --> StoreConfig
    StoreConfig --> FS
```

## Module integration sequence

```mermaid
sequenceDiagram
    actor User
    participant Update as update.go
    participant Create as create.go
    participant StoreCreate as store/create.go
    participant Ticket as internal/ticket
    participant FS as filesystem
    participant Editor as external editor

    User->>Update: press n (on board)
    Update->>Create: enterCreate()
    Create-->>User: render 4-field form

    User->>Update: fill fields + press enter
    Update->>Create: submitCreate()
    Create->>StoreCreate: Create(root, kind, title, labels, priority, now)
    StoreCreate->>Ticket: GenerateID(existingIDs)
    Ticket-->>StoreCreate: TC-XXXXXX
    StoreCreate->>Ticket: NewMarkdownFullWithID(...)
    Ticket-->>StoreCreate: markdown content
    StoreCreate->>FS: os.WriteFile(backlog/tc-xxxxxx-slug.md)
    StoreCreate-->>Create: path

    Create->>StoreCreate: LoadBoard(root)
    StoreCreate-->>Create: Board

    alt SkipEditorPrompt = false
        Create-->>User: render post-create dialog
        User->>Update: press y
        Update->>Editor: tea.ExecProcess(editor, path)
        Editor-->>Update: process exits
        Update->>Create: handleEditorFinished()
        Create->>StoreCreate: LoadBoard(root)
    else SkipEditorPrompt = true
        Create-->>User: success notification + board
    end
```
