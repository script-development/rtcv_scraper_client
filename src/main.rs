mod messages;
mod rtcv_types;

use messages::{InMessages, OutMessages};
use rtcv_types::{ApiKeyInfo, ErrorResponse, GetStatusResponse};
use serde::de::DeserializeOwned;
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
            let health_req: Result<GetStatusResponse, String> = get_request(url).await;
            if let Err(err) = health_req {
                return OutMessages::ErrorAuth(err.to_string());
            }

            let url = format!("{}/api/v1/auth/keyinfo", credentials.server_location);
            let key_info_req: Result<ApiKeyInfo, String> = get_request(url).await;
            if let Err(err) = key_info_req {
                return OutMessages::ErrorAuth(err.to_string());
            }

            OutMessages::Ok
        }
        InMessages::Ping => OutMessages::Pong,
    }
}

async fn get_request<T: DeserializeOwned>(uri: impl AsRef<str>) -> Result<T, String> {
    let mut res = surf::get(uri).await.map_err(|e| e.to_string())?;
    let body = res.body_string().await.map_err(|e| e.to_string())?;

    if res.status().is_server_error() || res.status().is_client_error() {
        let err_resp: ErrorResponse = serde_json::from_str(&body).map_err(|e| e.to_string())?;
        return Err(err_resp.error);
    }

    serde_json::from_str(&body).map_err(|e| e.to_string())
}
