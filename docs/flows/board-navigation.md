# Board Navigation

Column/row movement, multi-select, horizontal scroll, and vertical scroll within columns.

## User flow

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TD
    Start["TUI launches\nViewBoard + InteractionBoard"] --> Init["LoadBoard from disk\ndefault col=0 row=0"]
    Init --> Loop["Keyboard input loop"]

    Loop --> ColNav{"h / l\nor ← →"}
    ColNav --> MoveCol["moveColumn(±1)\nadjust SelectedCol"]
    MoveCol --> EnsureColVis["ensureColVisible()\nadjust ColScrollOffset\nif col out of view"]
    EnsureColVis --> Loop

    Loop --> RowNav{"j / k\nor ↓ ↑"}
    RowNav --> MoveRow["moveRow(±1)\nadjust SelectedRows[state]"]
    MoveRow --> EnsureRowVis["ensureSelectedVisible(state)\nline-budget scroll\nadjust ColumnScroll[state]"]
    EnsureRowVis --> Loop

    Loop --> PageNav{"d / u\npage"}
    PageNav --> PageRows["pageRows(±1)\nhalf visible rows"]
    PageRows --> EnsureRowVis

    Loop --> MultiSel{"v"}
    MultiSel --> Toggle["toggleSelection()\nMultiSelected[state][name]"]
    Toggle --> Loop

    Loop --> HScroll{"terminal\ntoo narrow"}
    HScroll --> HScrollLogic["visibleColumnCount = width/60\nColScrollOffset tracks\nfirst visible column"]
    HScrollLogic --> Loop
```

## Module architecture

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
graph LR
    subgraph TUI
        Update["update.go\nkey dispatch"]
        Nav["navigation.go\nmoveColumn, moveRow\nensureColVisible\nensureSelectedVisible\npageRows"]
        Actions["actions.go\ntoggleSelection"]
        Layout["layout.go\ncolumnWidth\ncolumnLineBudget\nvisibleColumnCount"]
        Render["render_board.go\ncolumn rendering\nscroll indicators"]
        Model["model.go\nSelectedCol, ColScrollOffset\nSelectedRows, ColumnScroll\nMultiSelected, Width/Height"]
    end

    Update --> Nav
    Update --> Actions
    Nav --> Layout
    Nav --> Model
    Actions --> Model
    Render --> Layout
    Render --> Model
```

## Module integration sequence

```mermaid
sequenceDiagram
    actor User
    participant Update as update.go
    participant Nav as navigation.go
    participant Layout as layout.go
    participant Model as Model state
    participant Render as render_board.go

    User->>Update: press h/l
    Update->>Nav: moveColumn(±1)
    Nav->>Model: adjust SelectedCol
    Nav->>Nav: ensureColVisible()
    Nav->>Layout: visibleColumnCount()
    Layout-->>Nav: count based on Width
    Nav->>Model: adjust ColScrollOffset

    User->>Update: press j/k
    Update->>Nav: moveRow(±1)
    Nav->>Model: adjust SelectedRows[state]
    Nav->>Nav: ensureSelectedVisible(state)
    Nav->>Layout: columnLineBudget()
    Layout-->>Nav: available lines
    Nav->>Model: adjust ColumnScroll[state]

    User->>Update: press v
    Update->>Nav: toggleSelection()
    Nav->>Model: toggle MultiSelected[state][name]

    Model->>Render: View() called
    Render->>Layout: columnWidth(), visibleColumnCount()
    Render-->>User: board with cursor + selection markers
```
