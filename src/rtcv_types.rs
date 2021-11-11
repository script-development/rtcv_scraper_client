use serde::Deserialize;

#[derive(Deserialize)]
pub struct GetStatusResponse {
    // status: bool,
// appVersion: String,
}

#[derive(Deserialize)]
pub struct ApiKeyInfo {
    // id: String,
    // domains: Vec<String>,
    pub roles: Vec<ApiRole>,
    // system: bool,
}

#[derive(Deserialize)]
pub struct ApiRole {
    pub role: u64,
    // slug: String,
    // description: String,
}

impl ApiRole {
    pub fn is_scraper(&self) -> bool {
        self.role == 1
    }
}

#[derive(Deserialize)]
pub struct ErrorResponse {
    pub error: String,
}
