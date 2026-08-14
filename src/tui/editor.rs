use std::path::{Path, PathBuf};

/// Return the editor binary + args to open `path`.
/// Resolution order: $EDITOR env → "vi".
/// Args are parsed as shell words so quoted args work (TUI-08).
pub fn editor_path(ticket: &Path) -> (PathBuf, Vec<String>) {
    let editor = std::env::var("EDITOR").unwrap_or_else(|_| "vi".to_owned());
    let mut parts = shell_words::split(&editor).unwrap_or_else(|_| vec!["vi".to_owned()]);
    if parts.is_empty() {
        parts.push("vi".to_owned());
    }
    let bin = PathBuf::from(parts.remove(0));
    let mut args = parts;
    args.push(ticket.to_string_lossy().into_owned());
    (bin, args)
}

/// Launch the editor synchronously; terminal must already be restored.
pub fn open(ticket: &Path) {
    let (bin, args) = editor_path(ticket);
    let _ = std::process::Command::new(&bin).args(&args).status();
}
