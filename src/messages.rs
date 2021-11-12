use crate::rtcv_types::UserSecret;
use serde::{Deserialize, Serialize};
use serde_json::{Error as JsonError, Value as JsonValue};

#[derive(Debug, Serialize)]
#[serde(tag = "type", content = "content", rename_all = "snake_case")]
pub enum OutMessages {
    Ready(String),
    Pong,
    Ok(OkContent),
    Error(String),
}

#[derive(Debug, Serialize)]
#[serde(untagged)]
pub enum OkContent {
    Empty,
    String(String),
    Secret(JsonValue),
    UserSecret(UserSecret),
    UsersSecret(Vec<UserSecret>),
}

impl Into<OutMessages> for OkContent {
    fn into(self) -> OutMessages {
        OutMessages::Ok(self)
    }
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
    SetCredentials(InSetCredentials),
    SendCv(JsonValue),
    GetSecret(InGetSecret),
    GetUsersSecret(InGetSecret),
    GetUserSecret(InGetSecret),
    Ping,
}

impl InMessages {
    pub fn from_json(s: &str) -> Result<Self, JsonError> {
        serde_json::from_str(s)
    }
}

#[derive(Debug, Deserialize)]
pub struct InSetCredentials {
    pub server_location: String, // http://localhost:4000
    pub api_key_id: String,
    pub api_key: String,
}

#[derive(Debug, Deserialize)]
pub struct InGetSecret {
    pub encryption_key: String,
    pub key: Option<String>,
}
