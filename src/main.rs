mod messages;
mod rtcv_types;

use core::fmt::Write;
use messages::{InCredentials, InMessages, OutMessages};
use rtcv_types::{ApiKeyInfo, ErrorResponse, GetStatusResponse};
use serde::de::DeserializeOwned;
use sha2::{Digest, Sha512};
use std::io;

#[async_std::main]
async fn main() -> std::io::Result<()> {
    OutMessages::Ready(String::from("waiting for credentials")).print();

    let mut buffer = String::new();

    loop {
        buffer.clear();
        io::stdin().read_line(&mut buffer).unwrap();
        let input = match InMessages::from_json(buffer.trim()) {
            Err(err) => {
                OutMessages::ErrorInvalidJsonInput(err.to_string()).print();
                continue;
            }
            Ok(v) => v,
        };

        handle_in_message(input).await.print();
    }
}

async fn handle_in_message(input: InMessages) -> OutMessages {
    match input {
        InMessages::Credentials(credentials) => {
            let url = format!("{}/api/v1/health", credentials.server_location);
            let health_req: Result<GetStatusResponse, String> = get_request(url, None).await;
            if let Err(err) = health_req {
                return OutMessages::ErrorAuth(err.to_string());
            }

            let url = format!("{}/api/v1/auth/keyinfo", credentials.server_location);
            let key_info_req: Result<ApiKeyInfo, String> =
                get_request(url, Some(credentials)).await;
            if let Err(err) = key_info_req {
                return OutMessages::ErrorAuth(err.to_string());
            }

            OutMessages::Ok
        }
        InMessages::Ping => OutMessages::Pong,
    }
}

async fn get_request<T: DeserializeOwned>(
    uri: impl AsRef<str>,
    credentials: Option<InCredentials>,
) -> Result<T, String> {
    let mut req = surf::get(uri).header("Content-Type", "application/json");
    if let Some(credentials) = credentials {
        let mut hasher = Sha512::new();
        hasher.update(credentials.api_key);
        let hash_result = hasher.finalize();

        let mut api_key_hashed = String::with_capacity(2 * hash_result.len());
        for byte in hash_result {
            write!(api_key_hashed, "{:02X}", byte).unwrap();
        }

        let authorization_value = format!("Basic {}:{}", credentials.api_key_id, api_key_hashed);
        req = req.header("Authorization", authorization_value);
    }

    let mut res = req.await.map_err(|e| e.to_string())?;
    let body = res.body_string().await.map_err(|e| e.to_string())?;

    if res.status().is_server_error() || res.status().is_client_error() {
        let err_resp: ErrorResponse = serde_json::from_str(&body).map_err(|e| e.to_string())?;
        return Err(err_resp.error);
    }

    serde_json::from_str(&body).map_err(|e| e.to_string())
}
