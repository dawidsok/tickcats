use std::collections::HashMap;
use std::error::Error;
use std::fmt;

use chrono::{DateTime, NaiveDate};

const ID_PREFIX: &str = "TC-";
const ID_ALPHABET: &str = "ABCDEFGHJKMNPQRSTUVWXYZ23456789";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParseError(String);

impl ParseError {
    fn new(message: impl Into<String>) -> Self {
        Self(message.into())
    }
}

impl fmt::Display for ParseError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl Error for ParseError {}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum Priority {
    P0,
    P1,
    P2,
    P3,
}

impl Priority {
    pub fn parse(raw: &str) -> Result<Self, ParseError> {
        match raw.trim().to_ascii_uppercase().as_str() {
            "P0" => Ok(Self::P0),
            "P1" => Ok(Self::P1),
            "P2" => Ok(Self::P2),
            "P3" => Ok(Self::P3),
            _ => Err(ParseError::new(format!("invalid priority {raw:?}"))),
        }
    }

    pub const fn rank(self) -> u8 {
        self as u8
    }
}

impl fmt::Display for Priority {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::P0 => "P0",
            Self::P1 => "P1",
            Self::P2 => "P2",
            Self::P3 => "P3",
        })
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    Feature,
    Task,
    Bug,
}

impl Kind {
    pub const fn prefix(self) -> &'static str {
        match self {
            Self::Feature => "Feat",
            Self::Task => "Task",
            Self::Bug => "Bug",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedTitle {
    pub raw: String,
    pub labels: Vec<String>,
    pub kind: Kind,
    pub text: String,
    pub had_prefix: bool,
}

impl ParsedTitle {
    pub fn parse(raw: &str) -> Self {
        let (labels, rest) = split_labels(raw.trim());
        let (kind, text, had_prefix) = split_kind(rest);
        Self {
            raw: raw.to_owned(),
            labels,
            kind,
            text: text.to_owned(),
            had_prefix,
        }
    }

    pub fn has_label(&self, label: &str) -> bool {
        let label = normalize_label(label);
        self.labels.iter().any(|candidate| candidate == &label)
    }

    pub fn blocked(&self) -> bool {
        self.has_label("blocked")
    }

    pub fn to_refine(&self) -> bool {
        self.has_label("to refine")
    }

    pub fn normalized(&self) -> String {
        let labels = if self.labels.is_empty() {
            String::new()
        } else {
            format!("[{}] ", self.labels.join(", "))
        };
        if self.text.is_empty() {
            format!("{labels}{}:", self.kind.prefix())
        } else {
            format!("{labels}{}: {}", self.kind.prefix(), self.text)
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Ticket {
    pub id: String,
    pub title: String,
    pub parsed_title: ParsedTitle,
    pub priority: Priority,
    pub created: DateTime<chrono::FixedOffset>,
    pub updated: DateTime<chrono::FixedOffset>,
    pub deadline: Option<NaiveDate>,
    pub important: bool,
    pub body: Vec<u8>,
    pub has_acceptance_criteria: bool,
}

pub fn parse_markdown(data: &[u8]) -> Result<Ticket, ParseError> {
    let normalized = normalize_crlf(data);
    let rest = normalized
        .strip_prefix(b"---\n")
        .ok_or_else(|| ParseError::new("missing frontmatter opening fence"))?;
    let end = rest
        .windows(b"\n---\n".len())
        .position(|window| window == b"\n---\n")
        .ok_or_else(|| ParseError::new("missing frontmatter closing fence"))?;
    let frontmatter = std::str::from_utf8(&rest[..end])
        .map_err(|error| ParseError::new(format!("frontmatter is not valid UTF-8: {error}")))?;
    let fields = parse_frontmatter(frontmatter)?;
    let body = &rest[end + b"\n---\n".len()..];

    let title = required(&fields, "title")?.trim();
    let priority = Priority::parse(required(&fields, "priority")?.trim())?;
    let created = parse_timestamp(&fields, "created")?;
    let updated = parse_timestamp(&fields, "updated")?;
    let deadline = parse_date(fields.get("deadline"), "deadline")?;
    let important = parse_bool(fields.get("important"), "important")?;

    Ok(Ticket {
        id: fields
            .get("id")
            .map_or("", String::as_str)
            .trim()
            .to_owned(),
        title: title.to_owned(),
        parsed_title: ParsedTitle::parse(title),
        priority,
        created,
        updated,
        deadline,
        important,
        body: body.to_owned(),
        has_acceptance_criteria: has_non_empty_section(body, "Acceptance Criteria"),
    })
}

pub fn valid_id(id: &str) -> bool {
    id.len() == ID_PREFIX.len() + 6
        && id.starts_with(ID_PREFIX)
        && id[ID_PREFIX.len()..]
            .chars()
            .all(|character| ID_ALPHABET.contains(character))
}

fn normalize_crlf(data: &[u8]) -> Vec<u8> {
    let mut normalized = Vec::with_capacity(data.len());
    let mut index = 0;
    while index < data.len() {
        if data[index..].starts_with(b"\r\n") {
            normalized.push(b'\n');
            index += 2;
        } else {
            normalized.push(data[index]);
            index += 1;
        }
    }
    normalized
}

fn parse_frontmatter(frontmatter: &str) -> Result<HashMap<String, String>, ParseError> {
    let mut fields = HashMap::new();
    for (index, raw_line) in frontmatter.lines().enumerate() {
        let line = raw_line.trim();
        if line.is_empty() {
            continue;
        }
        let (key, value) = line.split_once(':').ok_or_else(|| {
            ParseError::new(format!("invalid frontmatter line {}: {line:?}", index + 1))
        })?;
        let key = key.trim();
        if key.is_empty() {
            return Err(ParseError::new(format!(
                "invalid frontmatter line {}: empty key",
                index + 1
            )));
        }
        fields.insert(
            key.to_owned(),
            value.trim().trim_matches(['\'', '"']).to_owned(),
        );
    }
    Ok(fields)
}

fn required<'a>(fields: &'a HashMap<String, String>, key: &str) -> Result<&'a str, ParseError> {
    fields
        .get(key)
        .map(String::as_str)
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| ParseError::new(format!("missing required frontmatter field {key:?}")))
}

fn parse_timestamp(
    fields: &HashMap<String, String>,
    key: &str,
) -> Result<DateTime<chrono::FixedOffset>, ParseError> {
    let raw = required(fields, key)?.trim();
    DateTime::parse_from_rfc3339(raw)
        .map_err(|error| ParseError::new(format!("invalid {key} timestamp {raw:?}: {error}")))
}

fn parse_date(raw: Option<&String>, key: &str) -> Result<Option<NaiveDate>, ParseError> {
    let Some(raw) = raw
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
    else {
        return Ok(None);
    };
    NaiveDate::parse_from_str(raw, "%Y-%m-%d")
        .map(Some)
        .map_err(|error| {
            ParseError::new(format!(
                "invalid {key} date {raw:?}: expected YYYY-MM-DD: {error}"
            ))
        })
}

fn parse_bool(raw: Option<&String>, key: &str) -> Result<bool, ParseError> {
    let Some(raw) = raw
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
    else {
        return Ok(false);
    };
    match raw {
        "1" | "t" | "T" | "TRUE" | "true" | "True" => Ok(true),
        "0" | "f" | "F" | "FALSE" | "false" | "False" => Ok(false),
        _ => Err(ParseError::new(format!("invalid {key} bool {raw:?}"))),
    }
}

fn split_labels(raw: &str) -> (Vec<String>, &str) {
    if !raw.starts_with('[') {
        return (Vec::new(), raw);
    }
    let Some(end) = raw.find(']') else {
        return (Vec::new(), raw);
    };
    let labels = raw[1..end]
        .split(',')
        .map(normalize_label)
        .filter(|label| !label.is_empty())
        .collect();
    (labels, raw[end + 1..].trim())
}

fn split_kind(raw: &str) -> (Kind, &str, bool) {
    let Some((prefix, text)) = raw.split_once(':') else {
        return (Kind::Task, raw.trim(), false);
    };
    let kind = match prefix.trim().to_ascii_lowercase().as_str() {
        "feat" | "feature" => Kind::Feature,
        "bug" | "fix" => Kind::Bug,
        "task" => Kind::Task,
        _ => return (Kind::Task, raw.trim(), false),
    };
    (kind, text.trim(), true)
}

fn normalize_label(raw: &str) -> String {
    raw.trim().to_ascii_lowercase()
}

fn has_non_empty_section(markdown: &[u8], heading: &str) -> bool {
    let markdown = String::from_utf8_lossy(markdown);
    let wanted = format!("## {heading}");
    let mut in_section = false;
    for line in markdown.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with("## ") {
            if in_section {
                return false;
            }
            in_section = trimmed == wanted;
            continue;
        }
        if in_section
            && !trimmed.is_empty()
            && !trimmed
                .strip_prefix('-')
                .unwrap_or(trimmed)
                .trim()
                .is_empty()
        {
            return true;
        }
    }
    false
}
