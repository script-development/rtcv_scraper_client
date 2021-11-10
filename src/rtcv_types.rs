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
// roles: Vec<ApiRoles>,
// system: bool,
}

#[derive(Deserialize)]
pub struct ErrorResponse {
    pub error: String,
}
