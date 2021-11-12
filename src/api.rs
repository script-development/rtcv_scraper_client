use crate::messages::InSetCredentials;
use crate::rtcv_types::{ErrorResponse, UserSecret};
use core::fmt::Write;
use http_types::{Method, Url};
use serde::de::DeserializeOwned;
use serde::Serialize;
use sha2::{Digest, Sha512};
use std::collections::HashMap;
use std::time::SystemTime;
use surf::RequestBuilder;

pub struct Api {
    auth_header_value: String,
    credentials: Option<InSetCredentials>,
    pub cache: HashMap<String, SystemTime>,
}

impl Api {
    pub fn new() -> Self {
        Self {
            auth_header_value: String::new(),
            credentials: None,
            cache: HashMap::new(),
        }
    }

    pub fn set_credentials(&mut self, credentials: InSetCredentials) -> Result<(), String> {
        if credentials.server_location.len() == 0 {
            return Err("server_location cannot be empty".to_string());
        }
        if credentials.api_key_id.len() == 0 {
            return Err("api_key_id cannot be empty".to_string());
        }
        if credentials.api_key.len() == 0 {
            return Err("api_key cannot be empty".to_string());
        }

        let mut hasher = Sha512::new();
        hasher.update(credentials.api_key.as_str());
        let hash_result = hasher.finalize();

        let mut api_key_hashed = String::with_capacity(2 * hash_result.len());
        for byte in hash_result {
            write!(api_key_hashed, "{:02X}", byte).unwrap();
        }

        self.auth_header_value = format!(
            "Basic {}:{}",
            credentials.api_key_id.as_str(),
            api_key_hashed
        );
        self.credentials = Some(credentials);

        Ok(())
    }

    pub async fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T, String> {
        let body: Option<()> = None;
        self.do_request(Method::Get, path, body).await
    }

    pub async fn post<T: DeserializeOwned, Y: Serialize>(
        &self,
        path: &str,
        body: Option<Y>,
    ) -> Result<T, String> {
        self.do_request(Method::Post, path, body).await
    }

    pub async fn do_request<T: DeserializeOwned, Y: Serialize>(
        &self,
        method: Method,
        path: &str,
        body: Option<Y>,
    ) -> Result<T, String> {
        let server_location = match &self.credentials {
            None => return Err("credentials not set".to_string()),
            Some(v) => v.server_location.as_str(),
        };

        let uri: Url = format!("{}{}", server_location, path)
            .as_str()
            .parse()
            .unwrap();

        let mut req = RequestBuilder::new(method, uri)
            .header("Content-Type", "application/json")
            .header("Authorization", &self.auth_header_value);

        if let Some(body) = body {
            req = req.body_json(&body).map_err(|e| e.to_string())?;
        }

        let mut res = req.await.map_err(|e| e.to_string())?;

        let body = res.body_string().await.map_err(|e| e.to_string())?;

        if res.status().is_server_error() || res.status().is_client_error() {
            let err_resp: ErrorResponse = serde_json::from_str(&body).map_err(|e| e.to_string())?;
            return Err(err_resp.error);
        }

        serde_json::from_str(&body).map_err(|e| e.to_string())
    }

    pub async fn get_secret<T: DeserializeOwned>(
        &self,
        encryption_key: &str,
        key: &str,
    ) -> Result<T, String> {
        self.get(&format!("/api/v1/secrets/myKey/{}/{}", key, encryption_key))
            .await
    }

    pub async fn get_users_secret(
        &self,
        encryption_key: &str,
        key: Option<String>,
    ) -> Result<Vec<UserSecret>, String> {
        let key = key.unwrap_or(String::from("users"));

        let users_opt = self
            .get_secret::<Option<Vec<UserSecret>>>(encryption_key, &key)
            .await?;

        let users = match users_opt {
            Some(v) => v,
            None => return Err("no users found".to_string()),
        };
        if users.len() == 0 {
            return Err("no users found".to_string());
        }

        Ok(users)
    }

    pub async fn get_user_secret(
        &self,
        encryption_key: &str,
        key: Option<String>,
    ) -> Result<UserSecret, String> {
        let key = key.unwrap_or(String::from("user"));

        let user_opt = self
            .get_secret::<Option<UserSecret>>(encryption_key, &key)
            .await?;

        match user_opt {
            Some(v) => Ok(v),
            None => Err("no user found".to_string()),
        }
    }
}
