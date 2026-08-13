use std::error::Error;
use std::fmt;
use std::fs;
use std::path::{Path, PathBuf};

use serde_json::{Map, Value};

#[derive(Debug)]
pub struct ConfigError(String);

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl Error for ConfigError {}

#[derive(Debug, Clone)]
pub struct Config {
    path: PathBuf,
    value: Value,
}

impl Config {
    pub fn load(board_root: &Path) -> Result<Self, ConfigError> {
        let path = board_root.join("config.json");
        let value = match fs::read(&path) {
            Ok(data) => serde_json::from_slice(&data)
                .map_err(|error| ConfigError(format!("read {}: {error}", path.display())))?,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Value::Object(Map::new()),
            Err(error) => {
                return Err(ConfigError(format!("read {}: {error}", path.display())));
            }
        };
        validate_known_fields(&value)?;
        Ok(Self { path, value })
    }

    pub fn matrix_enabled(&self) -> bool {
        !self
            .value
            .get("disable_matrix_prioritisation")
            .and_then(Value::as_bool)
            .unwrap_or(false)
    }

    pub fn toggle_matrix(&mut self) -> Result<bool, ConfigError> {
        let enabled = !self.matrix_enabled();
        let object = self
            .value
            .as_object_mut()
            .ok_or_else(|| ConfigError("config must be a JSON object".to_owned()))?;
        if enabled {
            object.remove("disable_matrix_prioritisation");
        } else {
            object.insert(
                "disable_matrix_prioritisation".to_owned(),
                Value::Bool(true),
            );
        }
        Ok(enabled)
    }

    pub fn save(&self) -> Result<(), ConfigError> {
        let data = serde_json::to_vec_pretty(&self.value)
            .map_err(|error| ConfigError(format!("encode config: {error}")))?;
        fs::write(&self.path, data)
            .map_err(|error| ConfigError(format!("write {}: {error}", self.path.display())))
    }

    pub fn value(&self) -> &Value {
        &self.value
    }
}

fn validate_known_fields(value: &Value) -> Result<(), ConfigError> {
    let object = value
        .as_object()
        .ok_or_else(|| ConfigError("config must be a JSON object".to_owned()))?;
    validate_type(object, "editor", Value::is_string)?;
    validate_type(object, "theme", Value::is_i64)?;
    validate_type(object, "skip_editor_prompt", Value::is_boolean)?;
    validate_type(object, "disable_matrix_prioritisation", Value::is_boolean)?;

    if let Some(columns) = object.get("columns").filter(|columns| !columns.is_null()) {
        let columns = columns
            .as_array()
            .ok_or_else(|| ConfigError("config field \"columns\" has invalid type".to_owned()))?;
        for column in columns {
            if column.is_null() {
                continue;
            }
            let column = column
                .as_object()
                .ok_or_else(|| ConfigError("config column must be a JSON object".to_owned()))?;
            for key in ["id", "name", "color"] {
                if column
                    .get(key)
                    .is_some_and(|value| !value.is_null() && !value.is_string())
                {
                    return Err(ConfigError(format!(
                        "config column field {key:?} has invalid type"
                    )));
                }
            }
        }
    }
    Ok(())
}

fn validate_type(
    object: &Map<String, Value>,
    key: &str,
    predicate: impl Fn(&Value) -> bool,
) -> Result<(), ConfigError> {
    if object
        .get(key)
        .is_some_and(|value| !value.is_null() && !predicate(value))
    {
        return Err(ConfigError(format!(
            "config field {key:?} has invalid type"
        )));
    }
    Ok(())
}
