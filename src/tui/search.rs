/// fuzzy_match reports whether every char in `pattern` appears in `text` in
/// order (both must already be lowercased).
pub fn fuzzy_match(pattern: &str, text: &str) -> bool {
    let mut pi = pattern.chars();
    let mut want = match pi.next() {
        Some(c) => c,
        None => return true,
    };
    for c in text.chars() {
        if c == want {
            match pi.next() {
                Some(next) => want = next,
                None => return true,
            }
        }
    }
    false
}

/// Filter a slice of tickets by fuzzy query (searches priority + title + body).
pub fn filtered_tickets<'a>(
    tickets: &'a [crate::store::board::StoredTicket],
    query: &str,
) -> Vec<&'a crate::store::board::StoredTicket> {
    if query.is_empty() {
        return tickets.iter().collect();
    }
    let q = query.to_ascii_lowercase();
    tickets
        .iter()
        .filter(|t| {
            let body = String::from_utf8_lossy(&t.ticket.body);
            let hay =
                format!("{} {} {}", t.ticket.priority, t.ticket.title, body).to_ascii_lowercase();
            fuzzy_match(&q, &hay)
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fuzzy_match_subsequence() {
        assert!(fuzzy_match("tsk", "task: fix the thing"));
        assert!(fuzzy_match("p2", "p2 task: widget"));
        assert!(!fuzzy_match("xyz", "abc def"));
        assert!(fuzzy_match("", "anything")); // empty pattern always matches
    }
}
