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
pub struct SendCv {
    referenceNumber: String,
    educations: Option<Education>,
    courses: Option<Course>,
    workExperiences: Option<WorkExperience>,
    preferredJobs: Option<String>,
    languages: Option<Language>,
    personalDetails: Option<PersonalDetails>,
    driversLicenses: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PersonalDetails {
    initials: Option<String>,
    firstName: Option<String>,
    surNamePrefix: Option<String>,
    surName: Option<String>,
    dob: Option<String>,
    gender: Option<String>,
    streetName: Option<String>,
    houseNumber: Option<String>,
    houseNumberSuffix: Option<String>,
    zip: Option<String>,
    city: Option<String>,
    country: Option<String>,
    phoneNumber: Option<String>,
    email: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Education {
    name: Option<String>,
    description: Option<String>,
    institute: Option<String>,
    isCompleted: Option<bool>,
    hasDiploma: Option<bool>,
    startDate: Option<String>,
    endDate: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Course {
    name: Option<String>,
    institute: Option<String>,
    startDate: Option<String>,
    endDate: Option<String>,
    isCompleted: Option<bool>,
    description: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct WorkExperience {
    description: Option<String>,
    profession: Option<String>,
    startDate: Option<String>,
    endDate: Option<String>,
    stillEmployed: Option<bool>,
    employer: Option<String>,
    weeklyHoursWorked: Option<isize>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Language {
    name: String,
    levelSpoken: u8,
    levelWritten: u8,
}
