use std::collections::HashSet;
use std::ffi::OsString;
use std::path::PathBuf;

use crate::store::board::{Board, PickResult, State, StoredTicket, pick_next};
use crate::store::config::Config;
use crate::ticket::{Kind, Priority};

pub const STATES: [State; 4] = [State::Backlog, State::Ready, State::Wip, State::Done];
pub const KINDS: [Kind; 3] = [Kind::Feature, Kind::Task, Kind::Bug];
pub const PRIORITIES: [Priority; 4] = [Priority::P0, Priority::P1, Priority::P2, Priority::P3];

// --- Enums -------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq)]
pub struct CreateForm {
    pub kind: usize, // index into KINDS; default 0 = Feature
    pub title: String,
    pub cursor: usize,   // byte offset in title
    pub priority: usize, // index into PRIORITIES; default 2 = P2
    pub to_refine: bool, // default true
    pub field: usize,    // active field: 0=kind 1=title 2=priority 3=to_refine
    pub error: String,
}

impl Default for CreateForm {
    fn default() -> Self {
        Self {
            kind: 0,
            title: String::new(),
            cursor: 0,
            priority: 2,
            to_refine: true,
            field: 0,
            error: String::new(),
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct SearchState {
    pub query: String,
    pub typing: bool, // true = text entry; false = navigating results
}

impl Default for SearchState {
    fn default() -> Self {
        Self::new()
    }
}

impl SearchState {
    pub fn new() -> Self {
        Self {
            query: String::new(),
            typing: true,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub enum Mode {
    Board,
    Detail,
    Create(CreateForm),
    Matrix, // Eisenhower 2×2 grid view
}

#[derive(Debug, Clone, PartialEq)]
pub enum Overlay {
    None,
    DeleteConfirm,
    Help { scroll: usize },
}

// --- App struct --------------------------------------------------------------

#[derive(Debug)]
pub struct App {
    pub root: PathBuf,
    pub board: Board,
    pub config: Config,
    pub mode: Mode,
    pub overlay: Overlay,
    pub search: Option<SearchState>,
    pub col: usize,
    pub rows: [usize; 4],
    pub col_scroll: [usize; 4],
    pub col_offset: usize,
    pub detail_scroll: usize,
    pub detail_ticket: Option<OsString>,
    // Matrix view state
    pub matrix_quad: usize,       // focused quadrant 0-3
    pub matrix_rows: [usize; 4],  // cursor row per quadrant
    pub matrix_scroll: [usize; 4],// scroll offset per quadrant
    pub status: String,
    pub width: u16,
    pub height: u16,
}

impl App {
    pub fn new(root: PathBuf, board: Board, config: Config) -> Self {
        Self {
            root,
            board,
            config,
            mode: Mode::Board,
            overlay: Overlay::None,
            search: None,
            col: 0,
            rows: [0; 4],
            col_scroll: [0; 4],
            col_offset: 0,
            detail_scroll: 0,
            detail_ticket: None,
            matrix_quad: 0,
            matrix_rows: [0; 4],
            matrix_scroll: [0; 4],
            status: String::new(),
            width: 0,
            height: 0,
        }
    }

    // --- Board data helpers --------------------------------------------------

    pub fn col_tickets(&self, col: usize) -> &[StoredTicket] {
        self.board.tickets(STATES[col])
    }

    pub fn focused_ticket(&self) -> Option<&StoredTicket> {
        self.col_tickets(self.col).get(self.rows[self.col])
    }

    // --- Eisenhower matrix helpers ------------------------------------------

    /// Quadrant index for a ticket (0-3).
    /// Layout:  0=top-left(Schedule)  1=top-right(Do Now)
    ///          2=bottom-left(Later)  3=bottom-right(Delegate)
    /// [to refine] tickets always land in 2 (bottom-left).
    pub fn ticket_quadrant(ticket: &crate::ticket::Ticket) -> usize {
        use chrono::Utc;
        if ticket.parsed_title.to_refine() {
            return 2;
        }
        let now = Utc::now().date_naive();
        let urgent = ticket
            .deadline
            .is_some_and(|d| d.signed_duration_since(now).num_days() <= 7);
        match (ticket.important, urgent) {
            (true, false) => 0,  // Schedule
            (true, true)  => 1,  // Do Now
            (false, false) => 2, // Do Later
            (false, true)  => 3, // Delegate
        }
    }

    /// All non-Done tickets grouped into the 4 quadrants.
    pub fn matrix_quadrants(&self) -> [Vec<&StoredTicket>; 4] {
        let mut qs: [Vec<&StoredTicket>; 4] = Default::default();
        for col in 0..3 { // Backlog, Ready, WIP — skip Done
            for t in self.col_tickets(col) {
                qs[Self::ticket_quadrant(&t.ticket)].push(t);
            }
        }
        qs
    }

    /// Focused ticket in the current matrix quadrant.
    pub fn matrix_focused_ticket(&self) -> Option<&StoredTicket> {
        let qs = self.matrix_quadrants();
        qs[self.matrix_quad].get(self.matrix_rows[self.matrix_quad]).copied()
    }

    pub fn detail_ticket_ref(&self) -> Option<&StoredTicket> {
        let name = self.detail_ticket.as_ref()?;
        for col in 0..4 {
            if let Some(t) = self.col_tickets(col).iter().find(|t| &t.name == name) {
                return Some(t);
            }
        }
        None
    }

    pub fn recommended_names(&self) -> HashSet<OsString> {
        let mut set = HashSet::new();
        match pick_next(&self.board) {
            PickResult::One(t) => {
                set.insert(t.name);
            }
            PickResult::Tie(ts) => {
                for t in ts {
                    set.insert(t.name);
                }
            }
            PickResult::None => {}
        }
        set
    }

    pub fn matrix(&self) -> bool {
        self.config.matrix_enabled()
    }

    pub fn find_ticket(&self, name: &OsString) -> Option<(usize, usize)> {
        for col in 0..4 {
            if let Some(row) = self.col_tickets(col).iter().position(|t| &t.name == name) {
                return Some((col, row));
            }
        }
        None
    }

    // --- Layout helpers ------------------------------------------------------

    /// True when terminal is wide enough for a side-panel detail view.
    pub fn wide_detail(&self) -> bool {
        self.width >= 120
    }

    /// Number of columns visible simultaneously.
    pub fn visible_cols(&self) -> usize {
        if self.width == 0 {
            return 4;
        }
        ((self.width as usize) / 60).clamp(1, 4)
    }

    /// Width of the board area (may be narrowed when detail panel is open).
    pub fn board_area_width(&self) -> u16 {
        if matches!(self.mode, Mode::Detail) && self.wide_detail() {
            self.width * 60 / 100
        } else {
            self.width
        }
    }

    /// Width of one column cell (including borders).
    pub fn col_width(&self) -> u16 {
        let vis = self.visible_cols().max(1);
        let bw = self.board_area_width() as usize;
        ((bw / vis).saturating_sub(0).max(22)) as u16
    }

    /// Inner content width of a column.
    pub fn col_inner_width(&self) -> usize {
        (self.col_width() as usize).saturating_sub(4).max(1)
    }

    /// Height available for ticket rows in a column (excluding header+footer).
    pub fn col_body_height(&self) -> usize {
        if self.height == 0 {
            return 20;
        }
        // total - 3 (col borders+title) - 2 (footer)
        (self.height as usize).saturating_sub(5).max(4)
    }

    /// How many 2-line cards fit in the column body.
    pub fn visible_cards(&self) -> usize {
        // n cards occupy 2n + (n−1) = 3n−1 rows  →  n = (h+1)/3
        ((self.col_body_height() + 1) / 3).max(1)
    }

    // --- Cursor management ---------------------------------------------------

    pub fn ensure_visible(&mut self, col: usize) {
        let n = self.col_tickets(col).len();
        if n == 0 {
            self.col_scroll[col] = 0;
            return;
        }
        let row = self.rows[col].min(n - 1);
        let vis = self.visible_cards();
        if self.col_scroll[col] > row {
            self.col_scroll[col] = row;
        }
        if self.col_scroll[col] + vis <= row {
            self.col_scroll[col] = row + 1 - vis;
        }
    }

    pub fn clamp_cursor(&mut self) {
        self.col = self.col.min(3);
        for i in 0..4 {
            let n = self.col_tickets(i).len();
            if n == 0 {
                self.rows[i] = 0;
                self.col_scroll[i] = 0;
            } else {
                self.rows[i] = self.rows[i].min(n - 1);
                self.ensure_visible(i);
            }
        }
        let vis = self.visible_cols();
        if self.col_offset > self.col {
            self.col_offset = self.col;
        }
        if self.col_offset + vis <= self.col {
            self.col_offset = self.col + 1 - vis;
        }
        self.col_offset = self.col_offset.min(4usize.saturating_sub(vis));
    }

    pub fn move_col(&mut self, delta: i32) {
        self.col = (self.col as i32 + delta).clamp(0, 3) as usize;
        let vis = self.visible_cols();
        if self.col < self.col_offset {
            self.col_offset = self.col;
        }
        if self.col >= self.col_offset + vis {
            self.col_offset = self.col + 1 - vis;
        }
        self.col_offset = self.col_offset.min(4usize.saturating_sub(vis));
    }

    pub fn move_row(&mut self, delta: i32) {
        if let Some(s) = &self.search
            && !s.query.is_empty()
        {
            // Navigate within filtered results by name
            let filtered =
                crate::tui::search::filtered_tickets(self.col_tickets(self.col), &s.query);
            if filtered.is_empty() {
                return;
            }
            let names: Vec<OsString> = filtered.iter().map(|t| t.name.clone()).collect();
            let cur_name = self
                .col_tickets(self.col)
                .get(self.rows[self.col])
                .map(|t| t.name.clone());
            let fi = cur_name
                .as_ref()
                .and_then(|n| names.iter().position(|x| x == n))
                .unwrap_or(0);
            let new_fi = (fi as i32 + delta).clamp(0, names.len() as i32 - 1) as usize;
            let target = &names[new_fi];
            if let Some(i) = self
                .col_tickets(self.col)
                .iter()
                .position(|t| &t.name == target)
            {
                self.rows[self.col] = i;
            }
            self.ensure_visible(self.col);
            return;
        }
        let n = self.col_tickets(self.col).len();
        if n == 0 {
            return;
        }
        self.rows[self.col] = (self.rows[self.col] as i32 + delta).clamp(0, n as i32 - 1) as usize;
        self.ensure_visible(self.col);
    }

    pub fn half_page(&mut self, dir: i32) {
        let half = (self.visible_cards() / 2).max(1) as i32;
        self.move_row(dir * half);
    }

    // --- State reload --------------------------------------------------------

    pub fn reload(&mut self) -> bool {
        use crate::store::board::load;
        if let Ok(board) = load(&self.root) {
            self.board = board;
            self.config =
                Config::load(&self.root).unwrap_or_else(|_| Config::default_for(&self.root));
            self.clamp_cursor();
            true
        } else {
            false
        }
    }
}
