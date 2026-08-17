use std::ffi::OsString;
use std::path::PathBuf;

use chrono::Utc;
use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

use crate::store::operations::{create, move_ticket, set_important, trash};
use crate::tui::model::{App, CreateForm, KINDS, Mode, Overlay, PRIORITIES, STATES, SearchState};

#[derive(Debug, PartialEq)]
pub enum Action {
    Continue,
    Quit,
    Edit(PathBuf),
}

fn key_str(key: &KeyEvent) -> &'static str {
    match key.code {
        KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => "ctrl+c",
        KeyCode::Char('h') if key.modifiers.is_empty() => "h",
        KeyCode::Char('j') if key.modifiers.is_empty() => "j",
        KeyCode::Char('k') if key.modifiers.is_empty() => "k",
        KeyCode::Char('l') if key.modifiers.is_empty() => "l",
        KeyCode::Char('d') if key.modifiers.is_empty() => "d",
        KeyCode::Char('u') if key.modifiers.is_empty() => "u",
        KeyCode::Char('n') if key.modifiers.is_empty() => "n",
        KeyCode::Char('p') if key.modifiers.is_empty() => "p",
        KeyCode::Char('b') if key.modifiers.is_empty() => "b",
        KeyCode::Char('e') if key.modifiers.is_empty() => "e",
        KeyCode::Char('i') if key.modifiers.is_empty() => "i",
        KeyCode::Char('x') if key.modifiers.is_empty() => "x",
        KeyCode::Char('r') if key.modifiers.is_empty() => "r",
        KeyCode::Char('M') if key.modifiers.is_empty() || key.modifiers == KeyModifiers::SHIFT => "M",
        KeyCode::Char('q') if key.modifiers.is_empty() => "q",
        KeyCode::Char('y') if key.modifiers.is_empty() => "y",
        KeyCode::Char('/') if key.modifiers.is_empty() => "/",
        KeyCode::Char('?') => "?",
        KeyCode::Char(' ') => "space",
        KeyCode::Enter => "enter",
        KeyCode::Esc => "esc",
        KeyCode::Tab => "tab",
        KeyCode::BackTab => "shift+tab",
        KeyCode::Left => "left",
        KeyCode::Right => "right",
        KeyCode::Up => "up",
        KeyCode::Down => "down",
        KeyCode::Backspace => "backspace",
        KeyCode::Delete => "delete",
        _ => "",
    }
}

pub fn update(app: &mut App, key: KeyEvent) -> Action {
    // Global: Ctrl+C always quits from every mode
    if key.code == KeyCode::Char('c') && key.modifiers.contains(KeyModifiers::CONTROL) {
        return Action::Quit;
    }

    // Overlays take priority over mode
    match &app.overlay {
        Overlay::Help { .. } => return update_help(app, key),
        Overlay::DeleteConfirm => return update_delete_confirm(app, key),
        Overlay::None => {}
    }

    // Create form is its own full-screen mode
    if matches!(app.mode, Mode::Create(_)) {
        return update_create(app, key);
    }

    // Matrix view
    if matches!(app.mode, Mode::Matrix) {
        return update_matrix(app, key);
    }

    // Search overlay (active on top of Board)
    if let Some(s) = &app.search {
        if s.typing {
            return update_search_typing(app, key);
        } else {
            return update_search_nav(app, key);
        }
    }

    match &app.mode {
        Mode::Board => update_board(app, key),
        Mode::Detail => update_detail(app, key),
        Mode::Matrix | Mode::Create(_) => unreachable!(),
    }
}

// --- Board -------------------------------------------------------------------

fn update_board(app: &mut App, key: KeyEvent) -> Action {
    match key_str(&key) {
        "q" => return Action::Quit,
        "?" => {
            app.overlay = Overlay::Help { scroll: 0 };
        }
        "h" | "left" => app.move_col(-1),
        "l" | "right" => app.move_col(1),
        "j" | "down" => app.move_row(1),
        "k" | "up" => app.move_row(-1),
        "d" => app.half_page(1),
        "u" => app.half_page(-1),
        "enter" => open_detail(app),
        "n" => app.mode = Mode::Create(CreateForm::default()),
        "p" => do_move_ticket(app, 1),
        "b" => do_move_ticket(app, -1),
        "e" => return do_edit(app),
        "i" => do_toggle_important(app),
        "x" => enter_delete_confirm(app),
        "r" => {
            app.reload();
            app.status = "Reloaded".to_owned();
        }
        "M" => {
            app.mode = Mode::Matrix;
            app.status = String::new();
        }
        "/" => app.search = Some(SearchState::new()),
        _ => {}
    }
    Action::Continue
}

// --- Detail ------------------------------------------------------------------

fn update_detail(app: &mut App, key: KeyEvent) -> Action {
    match key_str(&key) {
        "q" => return Action::Quit,
        "?" => app.overlay = Overlay::Help { scroll: 0 },
        "esc" => {
            // Restore cursor to detail ticket's position before closing
            if let Some(name) = app.detail_ticket.take()
                && let Some((col, row)) = app.find_ticket(&name)
            {
                app.col = col;
                app.rows[col] = row;
                app.ensure_visible(col);
                app.move_col(0); // clamp col_offset
            }
            app.mode = Mode::Board;
            app.detail_scroll = 0;
        }
        "j" | "down" => app.detail_scroll = app.detail_scroll.saturating_add(1),
        "k" | "up" => app.detail_scroll = app.detail_scroll.saturating_sub(1),
        "d" => {
            let half = ((app.height as usize).saturating_sub(8) / 2).max(1);
            app.detail_scroll = app.detail_scroll.saturating_add(half);
        }
        "u" => {
            let half = ((app.height as usize).saturating_sub(8) / 2).max(1);
            app.detail_scroll = app.detail_scroll.saturating_sub(half);
        }
        "e" => return do_edit_detail(app),
        "i" => do_toggle_important_detail(app),
        _ => {}
    }
    Action::Continue
}

// --- Matrix view ------------------------------------------------------------

fn update_matrix(app: &mut App, key: KeyEvent) -> Action {
    match key_str(&key) {
        "q" => return Action::Quit,
        "M" | "esc" => {
            app.mode = Mode::Board;
            app.status = String::new();
        }
        "?" => app.overlay = Overlay::Help { scroll: 0 },
        "tab" => {
            // cycle quadrants clockwise: 0(top-left)→1(top-right)→3(bot-right)→2(bot-left)→0
            app.matrix_quad = match app.matrix_quad {
                0 => 1,
                1 => 3,
                3 => 2,
                _ => 0,
            };
        }
        "shift+tab" => {
            app.matrix_quad = match app.matrix_quad {
                0 => 2,
                2 => 3,
                3 => 1,
                _ => 0,
            };
        }
        "h" | "left" => {
            // move to left column (quadrants 0,2 are left; 1,3 are right)
            if app.matrix_quad == 1 { app.matrix_quad = 0; }
            if app.matrix_quad == 3 { app.matrix_quad = 2; }
        }
        "l" | "right" => {
            if app.matrix_quad == 0 { app.matrix_quad = 1; }
            if app.matrix_quad == 2 { app.matrix_quad = 3; }
        }
        "k" | "up" => {
            let q = app.matrix_quad;
            let n = app.matrix_quadrants()[q].len();
            if app.matrix_rows[q] > 0 {
                app.matrix_rows[q] -= 1;
            } else if q == 2 { app.matrix_quad = 0; }
              else if q == 3 { app.matrix_quad = 1; }
            let _ = n;
        }
        "j" | "down" => {
            let q = app.matrix_quad;
            let n = app.matrix_quadrants()[q].len();
            if n > 0 && app.matrix_rows[q] + 1 < n {
                app.matrix_rows[q] += 1;
            } else if q == 0 { app.matrix_quad = 2; }
              else if q == 1 { app.matrix_quad = 3; }
        }
        "enter" => open_detail_matrix(app),
        "e" => return do_edit_matrix(app),
        "i" => do_toggle_important_matrix(app),
        _ => {}
    }
    // Clamp cursor after navigation
    {
        let lens: [usize; 4] = {
            let qs = app.matrix_quadrants();
            [qs[0].len(), qs[1].len(), qs[2].len(), qs[3].len()]
        };
        for q in 0..4 {
            if lens[q] == 0 { app.matrix_rows[q] = 0; }
            else { app.matrix_rows[q] = app.matrix_rows[q].min(lens[q] - 1); }
        }
    }
    Action::Continue
}

fn open_detail_matrix(app: &mut App) {
    if let Some(t) = app.matrix_focused_ticket() {
        app.detail_ticket = Some(t.name.clone());
        app.mode = Mode::Detail;
        app.detail_scroll = 0;
    }
}

fn do_edit_matrix(app: &mut App) -> Action {
    if let Some(t) = app.matrix_focused_ticket() {
        Action::Edit(t.path.clone())
    } else {
        Action::Continue
    }
}

fn do_toggle_important_matrix(app: &mut App) {
    if let Some(t) = app.matrix_focused_ticket() {
        let name = t.name.clone();
        let state = t.state;
        let important = !t.ticket.important;
        use chrono::Utc;
        if crate::store::operations::set_important(&app.root, &name, state, important, Utc::now()).is_ok() {
            app.reload();
            app.status = if important { "Marked important".to_owned() } else { "Unmarked".to_owned() };
        }
    }
}

// --- Create form -------------------------------------------------------------

fn update_create(app: &mut App, key: KeyEvent) -> Action {
    let mut form = if let Mode::Create(f) = &app.mode {
        f.clone()
    } else {
        return Action::Continue;
    };

    let k = key_str(&key);
    match k {
        "esc" => {
            app.mode = Mode::Board;
            app.status = String::new();
            return Action::Continue;
        }
        "tab" => {
            form.field = (form.field + 1) % 4;
        }
        "shift+tab" => {
            form.field = (form.field + 3) % 4;
        }
        "enter" => {
            if form.title.trim().is_empty() {
                form.error = "Title required".to_owned();
                app.mode = Mode::Create(form);
                return Action::Continue;
            }
            // Submit
            let kind = KINDS[form.kind];
            let priority = PRIORITIES[form.priority];
            let title = form.title.trim().to_owned();
            let to_refine = form.to_refine;
            match create(
                &app.root,
                kind,
                &title,
                priority,
                to_refine,
                None,
                Utc::now(),
            ) {
                Ok(path) => {
                    let created_name: OsString = path.file_name().unwrap_or_default().to_owned();
                    app.reload();
                    // Focus the new ticket in backlog (col 0)
                    if let Some((col, row)) = app.find_ticket(&created_name) {
                        app.col = col;
                        app.rows[col] = row;
                        app.ensure_visible(col);
                        app.move_col(0);
                    }
                    app.mode = Mode::Board;
                    app.status = format!("Created {}", created_name.to_string_lossy());
                }
                Err(e) => {
                    form.error = format!("Create failed: {e}");
                    app.mode = Mode::Create(form);
                }
            }
            return Action::Continue;
        }
        _ => {}
    }

    // Field-specific handling
    match form.field {
        0 => match k {
            "h" | "left" => form.kind = (form.kind + KINDS.len() - 1) % KINDS.len(),
            "l" | "right" => form.kind = (form.kind + 1) % KINDS.len(),
            _ => {}
        },
        1 => {
            // Title text input
            match key.code {
                KeyCode::Char(c)
                    if key.modifiers.is_empty() || key.modifiers == KeyModifiers::SHIFT =>
                {
                    form.title.insert(form.cursor, c);
                    form.cursor += c.len_utf8();
                }
                KeyCode::Backspace => {
                    if form.cursor > 0 {
                        let prev = form.title[..form.cursor]
                            .char_indices()
                            .next_back()
                            .map(|(i, _)| i)
                            .unwrap_or(0);
                        form.title.drain(prev..form.cursor);
                        form.cursor = prev;
                    }
                }
                KeyCode::Delete => {
                    if form.cursor < form.title.len() {
                        let next = form.title[form.cursor..]
                            .char_indices()
                            .nth(1)
                            .map(|(i, _)| form.cursor + i)
                            .unwrap_or(form.title.len());
                        form.title.drain(form.cursor..next);
                    }
                }
                KeyCode::Left => {
                    if form.cursor > 0 {
                        form.cursor = form.title[..form.cursor]
                            .char_indices()
                            .next_back()
                            .map(|(i, _)| i)
                            .unwrap_or(0);
                    }
                }
                KeyCode::Right if form.cursor < form.title.len() => {
                    let next = form.title[form.cursor..]
                        .char_indices()
                        .nth(1)
                        .map(|(i, _)| form.cursor + i)
                        .unwrap_or(form.title.len());
                    form.cursor = next;
                }
                _ => {}
            }
            form.error.clear();
        }
        2 => match k {
            "h" | "left" => {
                form.priority = (form.priority + PRIORITIES.len() - 1) % PRIORITIES.len()
            }
            "l" | "right" => form.priority = (form.priority + 1) % PRIORITIES.len(),
            _ => {}
        },
        3 if (k == "space" || k == "h" || k == "l" || k == "left" || k == "right") => {
            form.to_refine = !form.to_refine;
        }
        _ => {}
    }

    app.mode = Mode::Create(form);
    Action::Continue
}

// --- Help --------------------------------------------------------------------

fn update_help(app: &mut App, key: KeyEvent) -> Action {
    let scroll = if let Overlay::Help { scroll } = app.overlay {
        scroll
    } else {
        0
    };
    match key_str(&key) {
        "?" | "esc" | "enter" => {
            app.overlay = Overlay::None;
        }
        "j" | "down" => {
            app.overlay = Overlay::Help { scroll: scroll + 1 };
        }
        "k" | "up" => {
            app.overlay = Overlay::Help {
                scroll: scroll.saturating_sub(1),
            };
        }
        "d" => {
            let half = ((app.height as usize) / 2).max(1);
            app.overlay = Overlay::Help {
                scroll: scroll + half,
            };
        }
        "u" => {
            let half = ((app.height as usize) / 2).max(1);
            app.overlay = Overlay::Help {
                scroll: scroll.saturating_sub(half),
            };
        }
        _ => {}
    }
    Action::Continue
}

// --- Delete confirm ----------------------------------------------------------

fn update_delete_confirm(app: &mut App, key: KeyEvent) -> Action {
    match key_str(&key) {
        "q" => return Action::Quit,
        "y" => {
            do_delete(app);
        }
        "n" | "esc" => {
            app.overlay = Overlay::None;
            app.status = String::new();
        }
        _ => {}
    }
    Action::Continue
}

// --- Search ------------------------------------------------------------------

fn update_search_typing(app: &mut App, key: KeyEvent) -> Action {
    let s = app.search.as_mut().unwrap();
    match key_str(&key) {
        "esc" => {
            app.search = None;
            return Action::Continue;
        }
        "enter" => {
            s.typing = false;
            return Action::Continue;
        }
        _ => {}
    }
    // Text input
    match key.code {
        KeyCode::Char(c) if key.modifiers.is_empty() || key.modifiers == KeyModifiers::SHIFT => {
            app.search.as_mut().unwrap().query.push(c);
        }
        KeyCode::Backspace => {
            app.search.as_mut().unwrap().query.pop();
        }
        _ => {}
    }
    Action::Continue
}

fn update_search_nav(app: &mut App, key: KeyEvent) -> Action {
    match key_str(&key) {
        "esc" => {
            if app.search.as_ref().is_none_or(|s| s.query.is_empty()) {
                app.search = None;
            } else {
                app.search.as_mut().unwrap().query.clear();
            }
        }
        "/" => {
            if let Some(s) = &mut app.search {
                s.typing = true;
            }
        }
        "j" | "down" => app.move_row(1),
        "k" | "up" => app.move_row(-1),
        "h" | "left" => app.move_col(-1),
        "l" | "right" => app.move_col(1),
        "enter" => open_detail(app),
        "q" => return Action::Quit,
        _ => {}
    }
    Action::Continue
}

// --- Action helpers ----------------------------------------------------------

fn open_detail(app: &mut App) {
    if let Some(t) = app.focused_ticket() {
        app.detail_ticket = Some(t.name.clone());
        app.mode = Mode::Detail;
        app.detail_scroll = 0;
    }
}

fn do_edit(app: &mut App) -> Action {
    if let Some(t) = app.focused_ticket() {
        let path = t.path.clone();
        Action::Edit(path)
    } else {
        Action::Continue
    }
}

fn do_edit_detail(app: &mut App) -> Action {
    if let Some(t) = app.detail_ticket_ref() {
        let path = t.path.clone();
        Action::Edit(path)
    } else {
        Action::Continue
    }
}

fn do_toggle_important(app: &mut App) {
    if let Some(t) = app.focused_ticket() {
        let name = t.name.clone();
        let state = STATES[app.col];
        let important = !t.ticket.important;
        if set_important(&app.root, &name, state, important, Utc::now()).is_ok() {
            app.reload();
            app.status = if important {
                "Marked important".to_owned()
            } else {
                "Unmarked".to_owned()
            };
        }
    }
}

fn do_toggle_important_detail(app: &mut App) {
    if let Some(t) = app.detail_ticket_ref() {
        let name = t.name.clone();
        let state = t.state;
        let important = !t.ticket.important;
        if set_important(&app.root, &name, state, important, Utc::now()).is_ok() {
            app.reload();
            app.status = if important {
                "Marked important".to_owned()
            } else {
                "Unmarked".to_owned()
            };
        }
    }
}

fn do_toggle_matrix(app: &mut App) {
    if let Ok(enabled) = app.config.toggle_matrix() {
        let _ = app.config.save();
        app.reload();
        app.status = if enabled {
            "Matrix: on".to_owned()
        } else {
            "Matrix: off".to_owned()
        };
    }
}

fn enter_delete_confirm(app: &mut App) {
    if let Some(t) = app.focused_ticket() {
        app.status = format!("Delete {}?", t.name.to_string_lossy());
        app.overlay = Overlay::DeleteConfirm;
    }
}

fn do_delete(app: &mut App) {
    let ticket = if let Some(t) = app.focused_ticket() {
        (t.name.clone(), STATES[app.col])
    } else {
        app.overlay = Overlay::None;
        return;
    };
    match trash(&app.root, &ticket.0, ticket.1) {
        Ok(_) => {
            app.overlay = Overlay::None;
            app.reload();
            app.status = format!("Deleted {}", ticket.0.to_string_lossy());
        }
        Err(e) => {
            app.overlay = Overlay::None;
            app.status = format!("Delete failed: {e}");
        }
    }
}

fn do_move_ticket(app: &mut App, direction: i32) {
    let ticket = if let Some(t) = app.focused_ticket() {
        (t.name.clone(), STATES[app.col])
    } else {
        app.status = "No ticket selected".to_owned();
        return;
    };
    let new_col = (app.col as i32 + direction).clamp(0, 3) as usize;
    if new_col == app.col {
        let edge = if direction > 0 {
            STATES[3].display()
        } else {
            STATES[0].display()
        };
        app.status = format!("Already in {edge}");
        return;
    }
    let to = STATES[new_col];
    match move_ticket(&app.root, &ticket.0, ticket.1, to) {
        Ok(_) => {
            app.reload();
            app.col = new_col;
            app.move_col(0);
            // Find ticket in new column
            if let Some((_, row)) = app.find_ticket(&ticket.0) {
                app.rows[new_col] = row;
                app.ensure_visible(new_col);
            }
            app.status = format!("Moved to {}", to.display());
        }
        Err(e) => {
            app.status = format!("Move failed: {e}");
        }
    }
}
