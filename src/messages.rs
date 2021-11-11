use crate::api::UserSecret;
use serde::{Deserialize, Serialize};
use serde_json::{Error as JsonError, Value as JsonValue};

#[derive(Debug, Serialize)]
#[serde(tag = "type", content = "content", rename_all = "snake_case")]
pub enum OutMessages {
    Ready(String),
    Pong,
    Ok,
    Secret(JsonValue),
    UserSecret(UserSecret),
    UsersSecret(Vec<UserSecret>),
    ErrorInvalidJsonInput(String),
    ErrorAuth(String),
    ErrorApi(String),
    ErrorInvalidInput(String),
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
    SendCv(SendCv),
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

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SendCv {
    reference_number: String,
    educations: Option<Education>,
    courses: Option<Course>,
    work_experiences: Option<WorkExperience>,
    preferred_jobs: Option<String>,
    languages: Option<Language>,
    personal_details: Option<PersonalDetails>,
    drivers_licenses: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PersonalDetails {
    initials: Option<String>,
    first_name: Option<String>,
    sur_name_prefix: Option<String>,
    sur_name: Option<String>,
    dob: Option<String>,
    gender: Option<String>,
    street_name: Option<String>,
    house_number: Option<String>,
    house_number_suffix: Option<String>,
    zip: Option<String>,
    city: Option<String>,
    country: Option<String>,
    phone_number: Option<String>,
    email: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Education {
    name: Option<String>,
    description: Option<String>,
    institute: Option<String>,
    is_completed: Option<bool>,
    has_diploma: Option<bool>,
    start_date: Option<String>,
    end_date: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Course {
    name: Option<String>,
    institute: Option<String>,
    start_date: Option<String>,
    end_date: Option<String>,
    is_completed: Option<bool>,
    description: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WorkExperience {
    description: Option<String>,
    profession: Option<String>,
    start_date: Option<String>,
    end_date: Option<String>,
    still_employed: Option<bool>,
    employer: Option<String>,
    weekly_hours_worked: Option<isize>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Language {
    name: String,
    level_spoken: u8,
    level_written: u8,
}
