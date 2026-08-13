//! Rust migration scaffold for TickCats.

/// Identifies the pre-port scaffold in tests and local builds.
pub const MIGRATION_SCAFFOLD: &str = "TickCats Rust migration scaffold";

#[cfg(test)]
mod tests {
    use super::MIGRATION_SCAFFOLD;

    #[test]
    fn scaffold_is_identifiable() {
        assert_eq!(MIGRATION_SCAFFOLD, "TickCats Rust migration scaffold");
    }
}
