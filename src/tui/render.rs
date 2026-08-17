use ratatui::Frame;
use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span, Text};
use ratatui::widgets::{Block, Borders, Clear, Paragraph};

use crate::store::board::StoredTicket;
use crate::tui::model::{App, CreateForm, KINDS, Mode, Overlay, PRIORITIES, STATES};
use crate::tui::search::filtered_tickets;

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

pub fn render(frame: &mut Frame, app: &mut App) {
    app.width = frame.area().width;
    app.height = frame.area().height;

    let area = frame.area();

    match &app.mode.clone() {
        Mode::Create(form) => render_create(frame, app, area, form),
        Mode::Matrix => render_matrix(frame, app, area),
        Mode::Board | Mode::Detail => render_board_or_detail(frame, app, area),
    }
}

// ---------------------------------------------------------------------------
// Board / detail layout
// ---------------------------------------------------------------------------

fn render_board_or_detail(frame: &mut Frame, app: &App, area: Rect) {
    let detail_open = matches!(app.mode, Mode::Detail);
    let wide = app.wide_detail();

    // Split area: [main_or_board | detail_panel] then footer
    let vertical = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(0), Constraint::Length(2)])
        .split(area);
    let main_area = vertical[0];
    let footer_area = vertical[1];

    if detail_open && wide {
        let horizontal = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(60), Constraint::Percentage(40)])
            .split(main_area);
        render_columns(frame, app, horizontal[0]);
        render_detail_panel(frame, app, horizontal[1]);
    } else if detail_open && !wide {
        render_detail_panel(frame, app, main_area);
    } else {
        render_columns(frame, app, main_area);
    }

    render_footer(frame, app, footer_area);

    // Overlays on top
    match &app.overlay {
        Overlay::Help { scroll } => render_help(frame, app, area, *scroll),
        Overlay::DeleteConfirm | Overlay::None => {}
    }
}

// ---------------------------------------------------------------------------
// Column board
// ---------------------------------------------------------------------------

fn render_columns(frame: &mut Frame, app: &App, area: Rect) {
    let vis = app.visible_cols().min(4);
    let start = app.col_offset.min(4usize.saturating_sub(vis));
    let end = (start + vis).min(4);

    // Search bar takes one line if active
    let (col_area, search_area) = if app.search.is_some() {
        let v = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Min(0), Constraint::Length(1)])
            .split(area);
        (v[0], Some(v[1]))
    } else {
        (area, None)
    };

    // Scroll indicator
    let (col_area, scroll_area) = if start > 0 || end < 4 {
        let v = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Length(1), Constraint::Min(0)])
            .split(col_area);
        (v[1], Some(v[0]))
    } else {
        (col_area, None)
    };

    // Column splits
    let constraints: Vec<Constraint> = (0..(end - start))
        .map(|_| Constraint::Ratio(1, (end - start) as u32))
        .collect();
    let col_areas = Layout::default()
        .direction(Direction::Horizontal)
        .constraints(constraints)
        .split(col_area);

    let recommended = app.recommended_names();
    for (i, col_idx) in (start..end).enumerate() {
        render_column(frame, app, col_areas[i], col_idx, &recommended);
    }

    if let (Some(area), true) = (scroll_area, start > 0 || end < 4) {
        let left: Vec<&str> = STATES[..start].iter().map(|s| s.display()).collect();
        let right: Vec<&str> = STATES[end..].iter().map(|s| s.display()).collect();
        let mut parts = Vec::new();
        if !left.is_empty() {
            parts.push(format!("← {}", left.join(", ")));
        }
        if !right.is_empty() {
            parts.push(format!("{} →", right.join(", ")));
        }
        let text = parts.join("  ");
        frame.render_widget(
            Paragraph::new(text).style(Style::default().fg(Color::DarkGray)),
            area,
        );
    }

    if let Some(area) = search_area {
        render_search_bar(frame, app, area);
    }
}

fn render_column(
    frame: &mut Frame,
    app: &App,
    area: Rect,
    col_idx: usize,
    recommended: &std::collections::HashSet<std::ffi::OsString>,
) {
    let focused = col_idx == app.col;
    let header_style = if focused {
        Style::default()
            .fg(Color::Blue)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::DarkGray)
    };
    let border_style = if focused {
        Style::default().fg(Color::Blue)
    } else {
        Style::default().fg(Color::DarkGray)
    };
    let title = STATES[col_idx].display().to_uppercase();
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(border_style)
        .title(Span::styled(format!(" {title} "), header_style));

    let inner = block.inner(area);
    frame.render_widget(block, area);
    render_column_tickets(frame, app, inner, col_idx, recommended);
}

fn render_column_tickets(
    frame: &mut Frame,
    app: &App,
    area: Rect,
    col_idx: usize,
    recommended: &std::collections::HashSet<std::ffi::OsString>,
) {
    let state = STATES[col_idx];
    let query = app.search.as_ref().map(|s| s.query.as_str()).unwrap_or("");
    let all_tickets = app.col_tickets(col_idx);
    let tickets: Vec<&StoredTicket> = if query.is_empty() {
        all_tickets.iter().collect()
    } else {
        filtered_tickets(all_tickets, query)
    };

    if tickets.is_empty() {
        frame.render_widget(
            Paragraph::new("  empty").style(Style::default().fg(Color::DarkGray)),
            area,
        );
        return;
    }

    let scroll = app.col_scroll[col_idx];
    let vis = app.visible_cards();
    let row_cursor = app.rows[col_idx];

    // Map full-list row_cursor → filtered row
    let filtered_cursor = if query.is_empty() {
        Some(row_cursor)
    } else {
        let cur_name = all_tickets.get(row_cursor).map(|t| &t.name);
        cur_name.and_then(|n| tickets.iter().position(|t| &t.name == n))
    };

    let mut lines: Vec<Line> = Vec::new();
    let inner_w = area.width as usize;

    // Scroll indicator above
    if scroll > 0 {
        lines.push(Line::styled(
            format!("  ↑ {scroll} above"),
            Style::default().fg(Color::DarkGray),
        ));
    }

    let separator = Line::styled(
        "─".repeat(inner_w.max(1)),
        Style::default().fg(Color::DarkGray),
    );

    let end = (scroll + vis + 1).min(tickets.len());
    for (i, ticket) in tickets[scroll..end].iter().enumerate() {
        let abs_i = scroll + i;
        let is_focused = col_idx == app.col && filtered_cursor == Some(abs_i);

        if i > 0 {
            lines.push(separator.clone());
        }

        let card = card_lines(
            ticket,
            app,
            col_idx,
            is_focused,
            recommended,
            inner_w,
            state,
        );
        lines.extend(card);
    }

    // Overflow indicator
    if end < tickets.len() {
        let below = tickets.len() - end;
        *lines.last_mut().unwrap() = Line::styled(
            format!("  ↓ {below} below"),
            Style::default().fg(Color::DarkGray),
        );
    }

    frame.render_widget(Paragraph::new(Text::from(lines)), area);
}

fn card_lines<'a>(
    ticket: &StoredTicket,
    app: &App,
    _col_idx: usize,
    focused: bool,
    recommended: &std::collections::HashSet<std::ffi::OsString>,
    width: usize,
    _state: crate::store::board::State,
) -> Vec<Line<'a>> {
    let focus_marker = if focused { ">" } else { " " };
    let star_marker = if recommended.contains(&ticket.name) {
        "★"
    } else {
        " "
    };
    let important_tag = if ticket.ticket.important {
        "[!]"
    } else {
        "   "
    };

    // Line 1: focus + star + important + title
    let raw_title = &ticket.ticket.title;
    let prefix = format!("{focus_marker} {star_marker} {important_tag} ");
    let avail = width.saturating_sub(prefix.chars().count()).max(1);
    let title_text = truncate_str(raw_title, avail);
    let line1_text = format!("{prefix}{title_text}");

    // Line 2: ID + badge + deadline
    let badge = if app.matrix() {
        let now = chrono::Utc::now().date_naive();
        let urgent = ticket
            .ticket
            .deadline
            .is_some_and(|d| d.signed_duration_since(now).num_days() <= 7);
        match (urgent, ticket.ticket.important) {
            (true, true) => "↑! ",
            (false, true) => "!  ",
            (true, false) => "↑  ",
            (false, false) => "   ",
        }
    } else {
        // When matrix off, show priority; no truncation needed (4 chars)
        // But we return as string below; handle specially
        ""
    };

    let id_str = if ticket.ticket.id.is_empty() {
        "—".to_owned()
    } else {
        ticket.ticket.id.clone()
    };
    let deadline_str = ticket
        .ticket
        .deadline
        .map(|d| format!("  {}", d))
        .unwrap_or_default();

    let line2_text = if app.matrix() {
        truncate_str(&format!("   {id_str}  {badge}{deadline_str}"), width).into_owned()
    } else {
        truncate_str(
            &format!("   {id_str}  {}{deadline_str}", ticket.ticket.priority),
            width,
        )
        .into_owned()
    };

    let style = if focused {
        Style::default().add_modifier(Modifier::BOLD)
    } else {
        Style::default()
    };
    let muted = Style::default().fg(Color::DarkGray);

    vec![
        Line::styled(line1_text, style),
        Line::styled(line2_text, muted),
    ]
}

// ---------------------------------------------------------------------------
// Detail panel
// ---------------------------------------------------------------------------

fn render_detail_panel(frame: &mut Frame, app: &App, area: Rect) {
    let ticket = match app.detail_ticket_ref() {
        Some(t) => t,
        None => {
            frame.render_widget(
                Paragraph::new("Ticket not found\n\nesc back")
                    .block(Block::default().borders(Borders::ALL).title(" Detail ")),
                area,
            );
            return;
        }
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::DarkGray))
        .title(Span::styled(" Detail ", Style::default().fg(Color::Blue)));
    let inner = block.inner(area);
    frame.render_widget(block, area);

    // Build content lines
    let mut lines: Vec<Line> = Vec::new();
    let w = inner.width as usize;

    // Metadata header
    let id_str = if ticket.ticket.id.is_empty() {
        "—".to_owned()
    } else {
        ticket.ticket.id.clone()
    };
    let now = chrono::Utc::now().date_naive();
    let urgent = ticket
        .ticket
        .deadline
        .is_some_and(|d| d.signed_duration_since(now).num_days() <= 7);
    let deadline_display = ticket
        .ticket
        .deadline
        .map(|d| {
            if urgent {
                format!("{d} (urgent!)")
            } else {
                d.to_string()
            }
        })
        .unwrap_or_else(|| "—".to_owned());

    lines.push(Line::styled(
        truncate_str(&ticket.ticket.title, w),
        Style::default().add_modifier(Modifier::BOLD),
    ));
    lines.push(Line::default());
    lines.push(Line::styled(
        format!("ID:       {id_str}"),
        Style::default().fg(Color::DarkGray),
    ));
    lines.push(Line::styled(
        format!("Priority: {}", ticket.ticket.priority),
        Style::default().fg(Color::DarkGray),
    ));
    lines.push(Line::styled(
        format!("State:    {}", ticket.state.display()),
        Style::default().fg(Color::DarkGray),
    ));
    let imp = if ticket.ticket.important { "yes" } else { "no" };
    lines.push(Line::styled(
        format!("Important:{imp:>4}"),
        Style::default().fg(Color::DarkGray),
    ));
    lines.push(Line::styled(
        format!("Deadline: {deadline_display}"),
        if urgent {
            Style::default().fg(Color::Red)
        } else {
            Style::default().fg(Color::DarkGray)
        },
    ));
    lines.push(Line::styled(
        format!("Created:  {}", ticket.ticket.created.format("%Y-%m-%d")),
        Style::default().fg(Color::DarkGray),
    ));
    lines.push(Line::styled(
        format!("Updated:  {}", ticket.ticket.updated.format("%Y-%m-%d")),
        Style::default().fg(Color::DarkGray),
    ));
    lines.push(Line::default());

    // Body
    let body = String::from_utf8_lossy(&ticket.ticket.body);
    for line in body.lines() {
        lines.push(Line::raw(truncate_str(line, w).into_owned()));
    }

    // Scrolling
    let scroll = app.detail_scroll.min(lines.len().saturating_sub(1));
    let visible: Vec<Line> = lines.into_iter().skip(scroll).collect();

    frame.render_widget(Paragraph::new(Text::from(visible)), inner);
}

// ---------------------------------------------------------------------------
// Footer
// ---------------------------------------------------------------------------

fn render_footer(frame: &mut Frame, app: &App, area: Rect) {
    let v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(1), Constraint::Length(1)])
        .split(area);
    let status_area = v[0];
    let hint_area = v[1];

    // Status line
    let status_text = if !app.status.is_empty() {
        app.status.as_str()
    } else {
        ""
    };
    frame.render_widget(
        Paragraph::new(status_text).style(Style::default().fg(Color::DarkGray)),
        status_area,
    );

    // Hint line
    let hint = hint_text(app);
    frame.render_widget(
        Paragraph::new(hint).style(Style::default().fg(Color::DarkGray)),
        hint_area,
    );
}

fn hint_text(app: &App) -> &'static str {
    if app.overlay == Overlay::DeleteConfirm {
        return "y confirm  n/esc cancel  q quit";
    }
    if matches!(app.overlay, Overlay::Help { .. }) {
        return "j/k scroll  ?/enter/esc close  q quit";
    }
    match &app.mode {
        Mode::Matrix => "h/l cols  j/k rows  tab quad  enter detail  e edit  i important  m board  q quit",
        Mode::Detail => "j/k scroll  e edit  i important  esc back  q quit",
        Mode::Create(_) => "tab field  h/l change  space toggle  enter create  esc cancel",
        Mode::Board => {
            if app.search.as_ref().is_some_and(|s| !s.typing) {
                "j/k tickets  h/l cols  enter detail  / edit  esc clear  q quit"
            } else if app.search.is_some() {
                "type query  enter navigate  esc cancel"
            } else {
                "h/l cols  j/k rows  enter detail  n new  p/b move  i !  x del  r reload  m matrix  / search  ? help  q"
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Search bar
// ---------------------------------------------------------------------------

fn render_search_bar(frame: &mut Frame, app: &App, area: Rect) {
    let s = match &app.search {
        Some(s) => s,
        None => return,
    };
    let cursor = if s.typing { "▌" } else { "" };
    let total: usize = (0..4)
        .map(|i| filtered_tickets(app.col_tickets(i), &s.query).len())
        .sum();
    let matches_str = if s.query.is_empty() {
        String::new()
    } else {
        format!("  {total} match(es)")
    };
    let text = format!("/ {}{cursor}{matches_str}", s.query);
    frame.render_widget(
        Paragraph::new(text).style(Style::default().fg(Color::DarkGray)),
        area,
    );
}

// ---------------------------------------------------------------------------
// Create form
// ---------------------------------------------------------------------------

fn render_create(frame: &mut Frame, _app: &App, area: Rect, form: &CreateForm) {
    let dialog_w = (area.width.saturating_sub(4)).clamp(44, 64);
    let dialog_h = 14u16;

    let x = (area.width.saturating_sub(dialog_w)) / 2;
    let y = (area.height.saturating_sub(dialog_h)) / 2;
    let dialog_area = Rect {
        x,
        y,
        width: dialog_w,
        height: dialog_h,
    };

    frame.render_widget(Clear, dialog_area);
    let block = Block::default().borders(Borders::ALL).title(" New Ticket ");
    let inner = block.inner(dialog_area);
    frame.render_widget(block, dialog_area);

    let inner_w = inner.width as usize;
    let label_w = 10usize;

    let kind_opts: String = KINDS
        .iter()
        .enumerate()
        .map(|(i, k)| {
            let s = format!("{k:?}");
            if i == form.kind {
                format!("[>{s}<]")
            } else {
                format!("[{s}]")
            }
        })
        .collect::<Vec<_>>()
        .join("  ");
    let pri_opts: String = PRIORITIES
        .iter()
        .enumerate()
        .map(|(i, p)| {
            if i == form.priority {
                format!("[>{p}<]")
            } else {
                format!("[{p}]")
            }
        })
        .collect::<Vec<_>>()
        .join("  ");
    let refine_box = if form.to_refine {
        "[x] to refine"
    } else {
        "[ ] to refine"
    };

    let field_style = |f: usize| -> Style {
        if f == form.field {
            Style::default()
                .fg(Color::Blue)
                .add_modifier(Modifier::BOLD)
        } else {
            Style::default()
        }
    };

    let mut lines: Vec<Line> = vec![
        Line::default(),
        Line::from(vec![
            Span::styled(format!("{:<label_w$}", "Kind"), field_style(0)),
            Span::raw(truncate_str(&kind_opts, inner_w.saturating_sub(label_w)).into_owned()),
        ]),
        Line::default(),
        Line::from(vec![
            Span::styled(format!("{:<label_w$}", "Title"), field_style(1)),
            Span::raw(format!(
                "{}{}",
                truncate_str(&form.title, inner_w.saturating_sub(label_w + 1)),
                if form.field == 1 { "▌" } else { "" }
            )),
        ]),
        Line::default(),
        Line::from(vec![
            Span::styled(format!("{:<label_w$}", "Priority"), field_style(2)),
            Span::raw(truncate_str(&pri_opts, inner_w.saturating_sub(label_w)).into_owned()),
        ]),
        Line::default(),
        Line::from(vec![
            Span::styled(format!("{:<label_w$}", "To Refine"), field_style(3)),
            Span::raw(refine_box),
        ]),
    ];
    if !form.error.is_empty() {
        lines.push(Line::default());
        lines.push(Line::styled(
            truncate_str(&form.error, inner_w).into_owned(),
            Style::default().fg(Color::Red),
        ));
    }

    frame.render_widget(Paragraph::new(Text::from(lines)), inner);
}

// ---------------------------------------------------------------------------
// Eisenhower matrix view
// ---------------------------------------------------------------------------

/// Quadrant metadata: (label, subtitle, index)
const QUADS: [(usize, usize, &str, &str); 4] = [
    (0, 0, "SCHEDULE",  "Important · Not Urgent"),
    (1, 0, "DO NOW",    "Important · Urgent"),
    (0, 1, "DO LATER",  "Not Important · Not Urgent"),
    (1, 1, "DELEGATE",  "Not Important · Urgent"),
];
// index mapping: quad 0=top-left 1=top-right 2=bottom-left 3=bottom-right

fn render_matrix(frame: &mut Frame, app: &App, area: Rect) {
    let vert = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(0), Constraint::Length(2)])
        .split(area);
    let grid_area = vert[0];
    let footer_area = vert[1];

    // Split grid into top and bottom rows
    let rows = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Ratio(1, 2), Constraint::Ratio(1, 2)])
        .split(grid_area);

    // Each row split left / right
    let top_cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Ratio(1, 2), Constraint::Ratio(1, 2)])
        .split(rows[0]);
    let bot_cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Ratio(1, 2), Constraint::Ratio(1, 2)])
        .split(rows[1]);

    // cell_area[quad_index]
    let cell_areas = [
        top_cols[0], // 0 top-left  Schedule
        top_cols[1], // 1 top-right Do Now
        bot_cols[0], // 2 bot-left  Do Later
        bot_cols[1], // 3 bot-right Delegate
    ];

    let qs = app.matrix_quadrants();

    for qi in 0..4 {
        let (_, _, label, subtitle) = QUADS[qi];
        let focused = qi == app.matrix_quad;
        let border_style = if focused {
            Style::default().fg(Color::Blue)
        } else {
            Style::default().fg(Color::DarkGray)
        };
        let title_style = if focused {
            Style::default().fg(Color::Blue).add_modifier(Modifier::BOLD)
        } else {
            Style::default().fg(Color::DarkGray)
        };
        let block = Block::default()
            .borders(Borders::ALL)
            .border_style(border_style)
            .title(Span::styled(
                format!(" {label} ({}) ", qs[qi].len()),
                title_style,
            ));
        let inner = block.inner(cell_areas[qi]);
        frame.render_widget(block, cell_areas[qi]);

        // Render subtitle + tickets
        let w = inner.width as usize;
        let mut lines: Vec<Line> = vec![
            Line::styled(truncate_str(subtitle, w).into_owned(),
                Style::default().fg(Color::DarkGray).add_modifier(Modifier::ITALIC)),
            Line::styled("─".repeat(w.max(1)), Style::default().fg(Color::DarkGray)),
        ];

        let tickets = &qs[qi];
        let scroll = app.matrix_scroll[qi];
        let visible_h = inner.height.saturating_sub(2) as usize; // subtract header lines
        let cursor = app.matrix_rows[qi];

        if tickets.is_empty() {
            lines.push(Line::styled("  (empty)", Style::default().fg(Color::DarkGray)));
        } else {
            let start = scroll.min(tickets.len().saturating_sub(1));
            if start > 0 {
                lines.push(Line::styled(
                    format!("  ↑ {start} above"),
                    Style::default().fg(Color::DarkGray),
                ));
            }
            let mut shown = 0usize;
            for (i, t) in tickets[start..].iter().enumerate() {
                let abs = start + i;
                let is_focused = focused && abs == cursor;
                let state_tag = match t.state {
                    crate::store::board::State::Backlog => "B",
                    crate::store::board::State::Ready   => "R",
                    crate::store::board::State::Wip     => "W",
                    crate::store::board::State::Done    => "D",
                };
                let imp = if t.ticket.important { "!" } else { " " };
                let dl = t.ticket.deadline
                    .map(|d| format!(" {d}"))
                    .unwrap_or_default();
                let prefix = if is_focused { ">" } else { " " };
                let line1 = truncate_str(
                    &format!("{prefix} [{state_tag}]{imp} {}", t.ticket.title),
                    w,
                ).into_owned();
                let line2 = truncate_str(
                    &format!("   {}  {}{dl}", t.ticket.id, t.ticket.priority),
                    w,
                ).into_owned();
                let style = if is_focused {
                    Style::default().add_modifier(Modifier::BOLD)
                } else {
                    Style::default()
                };
                let muted = Style::default().fg(Color::DarkGray);
                lines.push(Line::styled(line1, style));
                lines.push(Line::styled(line2, muted));
                shown += 2;
                if shown >= visible_h { break; }
            }
        }

        frame.render_widget(Paragraph::new(Text::from(lines)), inner);
    }

    render_footer(frame, app, footer_area);
}

// ---------------------------------------------------------------------------
// Help overlay
// ---------------------------------------------------------------------------

static HELP_LINES: &[&str] = &[
    "BOARD",
    "  h / ←        move column left",
    "  l / →        move column right",
    "  j / ↓        move row down",
    "  k / ↑        move row up",
    "  d / u        half-page down / up",
    "  enter        open detail panel",
    "  n            new ticket",
    "  p            progress → next column",
    "  b            move back ← previous column",
    "  e            edit in $EDITOR",
    "  i            toggle important",
    "  x            delete (soft) ticket",
    "  r            reload board",
    "  m            open Eisenhower matrix view",
    "  /            fuzzy search",
    "  ?            this help",
    "  q            quit",
    "",
    "MATRIX VIEW (M)",
    "  h / l        move between left/right column",
    "  j / k        move ticket cursor up/down",
    "  tab          cycle quadrants clockwise",
    "  shift+tab    cycle quadrants counter-clockwise",
    "  enter        open detail",
    "  e            edit in $EDITOR",
    "  i            toggle important (re-quadrants ticket)",
    "  m / esc      back to board",
    "",
    "  Quadrants:",
    "  top-left   SCHEDULE   important, not urgent",
    "  top-right  DO NOW     important, urgent (≤7 days)",
    "  bot-left   DO LATER   not important / [to refine]",
    "  bot-right  DELEGATE   not important, urgent",
    "",
    "DETAIL",
    "  j / k        scroll down / up",
    "  d / u        half-page",
    "  e            edit in $EDITOR",
    "  i            toggle important",
    "  esc          back to board",
    "",
    "SEARCH",
    "  /            enter or re-open query",
    "  enter        switch to navigation",
    "  esc          clear filter / exit",
    "",
    "CREATE",
    "  tab          next field",
    "  shift+tab    previous field",
    "  h / l        cycle option",
    "  space        toggle [to refine]",
    "  enter        create ticket",
    "  esc          cancel",
];

fn render_help(frame: &mut Frame, _app: &App, area: Rect, scroll: usize) {
    let dialog_w = (area.width.saturating_sub(6)).clamp(44, 60);
    let dialog_h = (area.height.saturating_sub(4))
        .min(HELP_LINES.len() as u16 + 2)
        .max(10);

    let x = (area.width.saturating_sub(dialog_w)) / 2;
    let y = (area.height.saturating_sub(dialog_h)) / 2;
    let dialog_area = Rect {
        x,
        y,
        width: dialog_w,
        height: dialog_h,
    };

    frame.render_widget(Clear, dialog_area);
    let block = Block::default()
        .borders(Borders::ALL)
        .title(" Keyboard Shortcuts ");
    let inner = block.inner(dialog_area);
    frame.render_widget(block, dialog_area);

    let visible_h = inner.height as usize;
    let max_scroll = HELP_LINES.len().saturating_sub(visible_h);
    let scroll = scroll.min(max_scroll);

    // Update scroll in app (we only have immutable ref here; hint line handles text)
    let lines: Vec<Line> = HELP_LINES
        .iter()
        .skip(scroll)
        .take(visible_h)
        .map(|&s| {
            if s.is_empty() {
                Line::default()
            } else if !s.starts_with(' ') {
                Line::styled(s, Style::default().add_modifier(Modifier::BOLD))
            } else {
                Line::raw(s)
            }
        })
        .collect();

    frame.render_widget(Paragraph::new(Text::from(lines)), inner);
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

fn truncate_str(s: &str, max_chars: usize) -> std::borrow::Cow<'_, str> {
    if max_chars == 0 {
        return std::borrow::Cow::Borrowed("");
    }
    let char_count = s.chars().count();
    if char_count <= max_chars {
        std::borrow::Cow::Borrowed(s)
    } else {
        let truncated: String = s.chars().take(max_chars.saturating_sub(1)).collect();
        std::borrow::Cow::Owned(format!("{truncated}…"))
    }
}
