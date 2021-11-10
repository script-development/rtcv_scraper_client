use serde::{Deserialize, Serialize};
use serde_json::Error as JsonError;

#[derive(Debug, Serialize)]
#[serde(tag = "type", content = "content", rename_all = "snake_case")]
pub enum OutMessages {
    Ready(String),
    Pong,
    Ok,
    ErrorInvalidJsonInput(String),
    ErrorAuth(String),
}

impl OutMessages {
    pub fn as_json(&self) -> String {
        serde_json::to_string(self).unwrap()
    }
    pub fn print(&self) {
        println!("{}", self.as_json());
    }
}

#[derive(Debug, Deserialize)]
#[serde(tag = "type", content = "content", rename_all = "snake_case")]
pub enum InMessages {
    Credentials(InCredentials),
    Ping,
}

impl InMessages {
    pub fn from_json(s: &str) -> Result<Self, JsonError> {
        serde_json::from_str(s)
    }
}

#[derive(Debug, Deserialize)]
pub struct InCredentials {
    pub server_location: String, // http://localhost:4000
    pub api_key_id: String,
    pub api_key: String,
}
