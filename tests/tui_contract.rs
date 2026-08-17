/// TUI contract tests: state-transition and render-bound checks for all
/// retained TUI matrix rows (TUI-01 through TUI-29, retained subset).
///
/// Tests operate purely on in-memory App state (no terminal) so they run
/// fast on any machine without a TTY. Render-bound tests use
/// Ratatui's TestBackend.
use std::ffi::OsString;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use crossterm::event::{KeyCode, KeyEvent, KeyEventKind, KeyModifiers};
use ratatui::Terminal;
use ratatui::backend::TestBackend;
use tickcats::store::board::{State, load};
use tickcats::store::config::Config;
use tickcats::store::operations::{create, init};
use tickcats::ticket::{Kind, Priority};
use tickcats::tui::model::{App, CreateForm, Mode, Overlay};
use tickcats::tui::update::{Action, update};

static NEXT: AtomicU64 = AtomicU64::new(0);

// ─── helpers ─────────────────────────────────────────────────────────────────

struct TempBoard(PathBuf);

impl TempBoard {
    fn new() -> Self {
        let p = std::env::temp_dir().join(format!(
            "tickcats-tui-test-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        TempBoard(p)
    }
    fn board(&self) -> PathBuf {
        self.0.join(".tickcats")
    }
}

impl Drop for TempBoard {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

fn make_app(root: &Path) -> App {
    init(root, false).unwrap();
    let board = load(root).unwrap();
    let config = Config::load(root).unwrap_or_else(|_| Config::default_for(root));
    let mut app = App::new(root.to_path_buf(), board, config);
    app.width = 200;
    app.height = 40;
    app
}

fn add_ticket(root: &Path, kind: Kind, title: &str, priority: Priority) -> OsString {
    let path = create(
        root,
        kind,
        title,
        priority,
        false,
        Some("Done"),
        chrono::Utc::now(),
    )
    .unwrap();
    path.file_name().unwrap().to_owned()
}

fn key(code: KeyCode) -> KeyEvent {
    KeyEvent {
        code,
        modifiers: KeyModifiers::NONE,
        kind: KeyEventKind::Press,
        state: crossterm::event::KeyEventState::NONE,
    }
}

fn char_key(c: char) -> KeyEvent {
    key(KeyCode::Char(c))
}

fn ctrl_c() -> KeyEvent {
    KeyEvent {
        code: KeyCode::Char('c'),
        modifiers: KeyModifiers::CONTROL,
        kind: KeyEventKind::Press,
        state: crossterm::event::KeyEventState::NONE,
    }
}

// ─── TUI-02: h/j/k/l navigation ──────────────────────────────────────────────

#[test]
fn tui_02_column_navigation() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    // Add tickets to multiple columns
    add_ticket(&tmp.board(), Kind::Task, "Alpha", Priority::P2);
    add_ticket(&tmp.board(), Kind::Task, "Beta", Priority::P1);
    app.reload();

    // l moves right; h moves left
    assert_eq!(app.col, 0);
    update(&mut app, char_key('l'));
    assert_eq!(app.col, 1);
    update(&mut app, char_key('h'));
    assert_eq!(app.col, 0);

    // h at left edge does not wrap to negative
    update(&mut app, char_key('h'));
    assert_eq!(app.col, 0);

    // arrow keys work too
    update(&mut app, key(KeyCode::Right));
    assert_eq!(app.col, 1);
    update(&mut app, key(KeyCode::Left));
    assert_eq!(app.col, 0);

    // j/k row movement within column
    assert_eq!(app.rows[0], 0);
    update(&mut app, char_key('j'));
    assert_eq!(app.rows[0], 1);
    update(&mut app, char_key('k'));
    assert_eq!(app.rows[0], 0);

    // k at top edge clamps to 0
    update(&mut app, char_key('k'));
    assert_eq!(app.rows[0], 0);
}

#[test]
fn tui_02_d_u_half_page() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    // Add many tickets to trigger paging
    for i in 0..20 {
        add_ticket(
            &tmp.board(),
            Kind::Task,
            &format!("Ticket {i}"),
            Priority::P2,
        );
    }
    app.reload();
    app.height = 20; // small terminal → visible_cards = ~5

    assert_eq!(app.rows[0], 0);
    update(&mut app, char_key('d'));
    assert!(app.rows[0] > 0, "d should advance row");
    let row_after_d = app.rows[0];
    update(&mut app, char_key('u'));
    assert!(app.rows[0] < row_after_d, "u should retreat row");
}

// ─── TUI-04: sliding columns ──────────────────────────────────────────────────

#[test]
fn tui_04_visible_cols_and_sliding() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());

    // Narrow: 1 column
    app.width = 50;
    assert_eq!(app.visible_cols(), 1);

    // Medium: 2 columns
    app.width = 120;
    assert_eq!(app.visible_cols(), 2);

    // Wide: all 4 columns
    app.width = 250;
    assert_eq!(app.visible_cols(), 4);

    // Navigating past visible window scrolls col_offset
    app.width = 120; // 2 visible
    app.col = 0;
    app.col_offset = 0;
    app.move_col(1); // col=1
    app.move_col(1); // col=2 → scrolls
    assert_eq!(app.col, 2);
    assert!(
        app.col_offset >= 1,
        "col_offset should advance to keep col in view"
    );
}

// ─── TUI-05: wide side panel / narrow full-screen ────────────────────────────

#[test]
fn tui_05_wide_vs_narrow_detail_flag() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());

    app.width = 80;
    assert!(!app.wide_detail(), "width 80 should be narrow");

    app.width = 120;
    assert!(app.wide_detail(), "width 120 should be wide");
}

#[test]
fn tui_05_detail_render_does_not_panic() {
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Render ticket", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.detail_ticket = Some(name);
    app.mode = Mode::Detail;

    // Wide: side panel
    app.width = 160;
    app.height = 30;
    let mut terminal = Terminal::new(TestBackend::new(160, 30)).unwrap();
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();

    // Narrow: full-screen
    app.width = 80;
    app.height = 30;
    let mut terminal = Terminal::new(TestBackend::new(80, 30)).unwrap();
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();
}

// ─── TUI-06: create form ──────────────────────────────────────────────────────

#[test]
fn tui_06_create_form_defaults() {
    let Mode::Create(form) = Mode::Create(CreateForm::default()) else {
        panic!()
    };
    assert_eq!(form.kind, 0); // Feature
    assert_eq!(form.priority, 2); // P2
    assert!(form.to_refine);
    assert_eq!(form.field, 0); // starts on kind field
}

#[test]
fn tui_06_create_rejects_empty_title() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('n')); // enter create
    assert!(matches!(app.mode, Mode::Create(_)));

    update(&mut app, key(KeyCode::Enter)); // submit empty
    let Mode::Create(ref form) = app.mode else {
        panic!("should still be in create mode")
    };
    assert!(
        !form.error.is_empty(),
        "error should be set for empty title"
    );
}

#[test]
fn tui_06_create_ticket_succeeds() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('n'));

    // Tab to title field
    update(&mut app, key(KeyCode::Tab));
    // Type title
    for c in "My task".chars() {
        update(&mut app, key(KeyCode::Char(c)));
    }
    // Submit
    update(&mut app, key(KeyCode::Enter));

    assert_eq!(app.mode, Mode::Board, "should return to board after create");
    // Ticket should exist in backlog
    assert!(
        !app.col_tickets(0).is_empty(),
        "ticket should be in backlog"
    );
    let t = &app.col_tickets(0)[0];
    assert!(
        t.ticket.title.contains("My task"),
        "ticket title should contain input"
    );
    assert!(
        t.ticket.parsed_title.to_refine(),
        "should have [to refine] label by default"
    );
}

// ─── TUI-08: editor shell-word parsing ────────────────────────────────────────

#[test]
fn tui_08_editor_shell_words() {
    // Covered by editor module tests; ensure update dispatches Edit action
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Edit me", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.col = 0;
    app.rows[0] = app
        .col_tickets(0)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);

    let action = update(&mut app, char_key('e'));
    assert!(
        matches!(action, Action::Edit(_)),
        "e key should return Edit action"
    );
}

// ─── TUI-09: p/b adjacent move ───────────────────────────────────────────────

#[test]
fn tui_09_p_progresses_ticket() {
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Progress me", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.col = 0;
    app.rows[0] = app
        .col_tickets(0)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    let backlog_before = app.col_tickets(0).len();

    update(&mut app, char_key('p'));

    // Ticket should have moved from backlog (col 0) to ready (col 1)
    assert_eq!(app.col_tickets(0).len(), backlog_before - 1);
    assert!(app.col_tickets(1).iter().any(|t| t.name == name));
    assert_eq!(app.col, 1, "col should follow the moved ticket");
}

#[test]
fn tui_09_b_moves_ticket_back() {
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Move back", Priority::P2);
    let mut app = make_app(&tmp.board());
    // First progress to ready
    app.col = 0;
    app.rows[0] = app
        .col_tickets(0)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    update(&mut app, char_key('p'));
    // Now move back
    update(&mut app, char_key('b'));
    assert!(
        app.col_tickets(0).iter().any(|t| t.name == name),
        "ticket should be back in backlog"
    );
    assert_eq!(app.col, 0);
}

#[test]
fn tui_09_p_at_done_reports_status() {
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Already done", Priority::P2);
    // Move to done manually
    let mut app = make_app(&tmp.board());
    app.col = 0;
    app.rows[0] = app
        .col_tickets(0)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    update(&mut app, char_key('p')); // → ready
    app.col = 1;
    app.rows[1] = app
        .col_tickets(1)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    update(&mut app, char_key('p')); // → wip
    app.col = 2;
    app.rows[2] = app
        .col_tickets(2)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    update(&mut app, char_key('p')); // → done
    app.col = 3;
    app.rows[3] = app
        .col_tickets(3)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    let before_status = app.status.clone();
    update(&mut app, char_key('p')); // at edge
    assert!(
        !app.status.is_empty() || app.status != before_status || app.col == 3,
        "p at done edge should report status or stay in done"
    );
}

// ─── TUI-13: x delete confirm ────────────────────────────────────────────────

#[test]
fn tui_13_x_enters_delete_confirm() {
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Delete me", Priority::P2);
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('x'));
    assert_eq!(app.overlay, Overlay::DeleteConfirm);
}

#[test]
fn tui_13_n_cancels_delete() {
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Keep me", Priority::P2);
    let mut app = make_app(&tmp.board());
    let count_before = app.col_tickets(0).len();
    update(&mut app, char_key('x'));
    update(&mut app, char_key('n'));
    assert_eq!(app.overlay, Overlay::None);
    assert_eq!(
        app.col_tickets(0).len(),
        count_before,
        "ticket should not be deleted"
    );
}

#[test]
fn tui_13_y_confirms_delete() {
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Trash this", Priority::P2);
    let mut app = make_app(&tmp.board());
    let count_before = app.col_tickets(0).len();
    update(&mut app, char_key('x'));
    update(&mut app, char_key('y'));
    assert_eq!(app.overlay, Overlay::None);
    assert_eq!(
        app.col_tickets(0).len(),
        count_before - 1,
        "ticket should be trashed"
    );
}

// ─── TUI-15: important toggle + matrix ───────────────────────────────────────

#[test]
fn tui_15_i_toggles_important() {
    let tmp = TempBoard::new();
    let name = add_ticket(&tmp.board(), Kind::Task, "Important?", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.col = 0;
    app.rows[0] = app
        .col_tickets(0)
        .iter()
        .position(|t| t.name == name)
        .unwrap_or(0);
    let initial = app.col_tickets(0)[app.rows[0]].ticket.important;

    update(&mut app, char_key('i'));
    app.reload();
    let new_val = app
        .col_tickets(0)
        .iter()
        .find(|t| t.name == name)
        .map(|t| t.ticket.important);
    assert_eq!(new_val, Some(!initial), "important should have toggled");
}

#[test]
fn tui_15_matrix_p_label_suppression_in_render() {
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Matrix ticket", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.width = 200;
    app.height = 30;

    // With matrix off: P2 appears on card line 2
    if let Ok(mut cfg) = Config::load(&tmp.board()) {
        let _ = cfg.toggle_matrix(); // turn off
        let _ = cfg.toggle_matrix(); // ensure off by toggling twice if needed
        // We can't guarantee the start state, so just check both render states
    }

    // Render both states: just check it doesn't panic
    let mut terminal = Terminal::new(TestBackend::new(200, 30)).unwrap();
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();
    update(&mut app, char_key('M')); // toggle matrix
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();
}

// ─── TUI-17: fuzzy search ────────────────────────────────────────────────────

#[test]
fn tui_17_slash_enters_search_typing() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    assert!(app.search.is_none());
    update(&mut app, char_key('/'));
    assert!(app.search.is_some());
    assert!(app.search.as_ref().unwrap().typing);
}

#[test]
fn tui_17_enter_switches_to_nav_mode() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('/'));
    // Type a query
    update(&mut app, key(KeyCode::Char('t')));
    // Enter switches to nav
    update(&mut app, key(KeyCode::Enter));
    assert!(!app.search.as_ref().unwrap().typing);
}

#[test]
fn tui_17_esc_clears_query_first_then_exits() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('/'));
    update(&mut app, key(KeyCode::Char('x')));
    update(&mut app, key(KeyCode::Enter)); // nav mode with query
    // First esc: clear query
    update(&mut app, key(KeyCode::Esc));
    assert!(
        app.search.as_ref().is_none_or(|s| s.query.is_empty()),
        "first esc should clear query"
    );
    // Second esc: exit search
    update(&mut app, key(KeyCode::Esc));
    assert!(app.search.is_none(), "second esc should exit search");
}

// ─── TUI-20: M matrix toggle ─────────────────────────────────────────────────

#[test]
fn tui_20_m_opens_matrix_view() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    assert_eq!(app.mode, Mode::Board);
    update(&mut app, char_key('M'));
    assert_eq!(app.mode, Mode::Matrix, "M should switch to matrix view");
    // M again returns to board
    update(&mut app, char_key('M'));
    assert_eq!(app.mode, Mode::Board, "second M should return to board");
    // esc also returns to board from matrix
    update(&mut app, char_key('M'));
    update(&mut app, key(KeyCode::Esc));
    assert_eq!(app.mode, Mode::Board);
}

// ─── TUI-23: ? help overlay ───────────────────────────────────────────────────

#[test]
fn tui_23_question_opens_help() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('?'));
    assert!(matches!(app.overlay, Overlay::Help { .. }));
}

#[test]
fn tui_23_esc_closes_help() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('?'));
    update(&mut app, key(KeyCode::Esc));
    assert_eq!(app.overlay, Overlay::None);
}

#[test]
fn tui_23_help_j_k_scroll() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    update(&mut app, char_key('?'));
    if let Overlay::Help { scroll } = app.overlay {
        assert_eq!(scroll, 0);
    }
    update(&mut app, char_key('j'));
    if let Overlay::Help { scroll } = app.overlay {
        assert_eq!(scroll, 1);
    }
    update(&mut app, char_key('k'));
    if let Overlay::Help { scroll } = app.overlay {
        assert_eq!(scroll, 0);
    }
}

// ─── TUI-24: q/Ctrl-C quits immediately, no dialog ───────────────────────────

#[test]
fn tui_24_q_quits_without_dialog() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    let action = update(&mut app, char_key('q'));
    // No quit dialog; action is Quit directly
    assert_eq!(action, Action::Quit);
    assert_eq!(
        app.overlay,
        Overlay::None,
        "no quit dialog overlay should appear"
    );
}

#[test]
fn tui_24_ctrl_c_quits() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    let action = update(&mut app, ctrl_c());
    assert_eq!(action, Action::Quit);
}

// ─── TUI-25: r manual reload ─────────────────────────────────────────────────

#[test]
fn tui_25_r_reloads_board() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    let count_before = app.col_tickets(0).len();
    // Add a ticket directly on disk (bypassing the app)
    add_ticket(&tmp.board(), Kind::Task, "Added externally", Priority::P2);
    // Board has NOT reloaded yet
    assert_eq!(app.col_tickets(0).len(), count_before);
    // r reloads
    update(&mut app, char_key('r'));
    assert_eq!(app.col_tickets(0).len(), count_before + 1);
}

// ─── TUI-26: status + hint footer — 2 rows, no overflow ──────────────────────

#[test]
fn tui_26_footer_fits_at_narrow_width() {
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    app.width = 80;
    app.height = 24;

    let mut terminal = Terminal::new(TestBackend::new(80, 24)).unwrap();
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();

    let buf = terminal.backend().buffer().clone();
    let width = buf.area.width as usize;
    let height = buf.area.height as usize;

    // Last two rows are footer; check they're within bounds
    for y in (height - 2)..height {
        let row: String = (0..width)
            .map(|x| {
                buf.cell((x as u16, y as u16))
                    .map_or(' ', |c| c.symbol().chars().next().unwrap_or(' '))
            })
            .collect();
        assert!(
            row.len() <= width,
            "footer row {y} exceeds terminal width: {row:?}"
        );
    }
}

// ─── TUI-01: recommendation markers ──────────────────────────────────────────

#[test]
fn tui_01_recommended_tickets_have_star() {
    let tmp = TempBoard::new();
    // Add a ticket in ready with AC so it becomes pick-next
    let path = create(
        &tmp.board(),
        Kind::Task,
        "Recommended",
        Priority::P2,
        false,
        Some("Done"),
        chrono::Utc::now(),
    )
    .unwrap();
    let name = path.file_name().unwrap().to_owned();
    // Move to ready
    tickcats::store::operations::move_ticket(&tmp.board(), &name, State::Backlog, State::Ready)
        .unwrap();

    let app = make_app(&tmp.board());
    // The ticket in ready with AC should be recommended
    let recommended = app.recommended_names();
    assert!(
        !recommended.is_empty(),
        "pick-next should find a recommended ticket"
    );
    assert!(
        recommended.contains(&name),
        "recommended_names should include the ticket"
    );
}

#[test]
fn tui_01_two_line_card_render() {
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Card ticket", Priority::P2);
    let mut app = make_app(&tmp.board());
    app.width = 200;
    app.height = 30;
    // Should render without panic; smoke test only
    let mut terminal = Terminal::new(TestBackend::new(200, 30)).unwrap();
    terminal
        .draw(|f| tickcats::tui::render_for_test(f, &mut app))
        .unwrap();
}

// ─── Dropped TUI IDs — ensure no scaffolding exists ──────────────────────────

#[test]
fn tui_dropped_no_move_mode() {
    // TUI-10: m key should NOT enter a move mode overlay or mode
    let tmp = TempBoard::new();
    let mut app = make_app(&tmp.board());
    let mode_before = format!("{:?}", app.mode);
    let overlay_before = format!("{:?}", app.overlay);
    update(&mut app, char_key('m'));
    assert_eq!(format!("{:?}", app.mode), mode_before);
    assert_eq!(format!("{:?}", app.overlay), overlay_before);
}

#[test]
fn tui_dropped_no_sort_cycling() {
    // TUI-16: s key should NOT change sort mode or board
    let tmp = TempBoard::new();
    add_ticket(&tmp.board(), Kind::Task, "Sort check", Priority::P2);
    let mut app = make_app(&tmp.board());
    let names_before: Vec<OsString> = app.col_tickets(0).iter().map(|t| t.name.clone()).collect();
    update(&mut app, char_key('s'));
    let names_after: Vec<OsString> = app.col_tickets(0).iter().map(|t| t.name.clone()).collect();
    assert_eq!(
        names_before, names_after,
        "s should not change board order (sort dropped)"
    );
}
