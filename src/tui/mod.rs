pub mod editor;
pub mod model;
pub mod render;
pub mod search;
pub mod update;

use std::io::stdout;
use std::path::Path;

use crossterm::ExecutableCommand;
use crossterm::event::{self, Event, KeyEventKind};
use crossterm::terminal::{
    EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode,
};
use ratatui::Terminal;
use ratatui::backend::CrosstermBackend;

use crate::store::board::{StoreError, load};
use crate::store::config::Config;

pub use model::{App, Mode, Overlay};

/// Thin wrapper for use in integration tests (avoids touching terminal I/O).
pub fn render_for_test(f: &mut ratatui::Frame, app: &mut App) {
    render::render(f, app);
}

pub fn run(root: &Path) -> Result<(), StoreError> {
    let board = load(root)?;
    let config = Config::load(root).unwrap_or_else(|_| Config::default_for(root));
    let mut app = App::new(root.to_path_buf(), board, config);

    enable_raw_mode().map_err(|e| StoreError::new(e.to_string()))?;
    stdout()
        .execute(EnterAlternateScreen)
        .map_err(|e| StoreError::new(e.to_string()))?;

    let backend = CrosstermBackend::new(stdout());
    let mut terminal = Terminal::new(backend).map_err(|e| StoreError::new(e.to_string()))?;

    let result = event_loop(&mut terminal, &mut app);

    // Always restore terminal even on error
    let _ = disable_raw_mode();
    let _ = stdout().execute(LeaveAlternateScreen);

    result
}

fn event_loop<B: ratatui::backend::Backend>(
    terminal: &mut Terminal<B>,
    app: &mut App,
) -> Result<(), StoreError> {
    loop {
        terminal
            .draw(|f| render::render(f, app))
            .map_err(|e| StoreError::new(e.to_string()))?;

        let ev = event::read().map_err(|e| StoreError::new(e.to_string()))?;
        match ev {
            Event::Key(key) if key.kind == KeyEventKind::Press => {
                match update::update(app, key) {
                    update::Action::Quit => break,
                    update::Action::Edit(path) => {
                        // Suspend TUI, run editor, resume
                        let _ = disable_raw_mode();
                        let _ = stdout().execute(LeaveAlternateScreen);
                        editor::open(&path);
                        let _ = enable_raw_mode();
                        let _ = stdout().execute(EnterAlternateScreen);
                        terminal.clear().ok();
                        app.reload();
                    }
                    update::Action::Continue => {}
                }
            }
            Event::Resize(w, h) => {
                app.width = w;
                app.height = h;
                app.clamp_cursor();
            }
            _ => {}
        }
    }
    Ok(())
}
