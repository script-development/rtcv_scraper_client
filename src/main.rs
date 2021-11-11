pub mod api;
pub mod messages;
pub mod rtcv_types;

use api::Api;
use messages::{InMessages, OutMessages};
use rtcv_types::{ApiKeyInfo, GetStatusResponse, ScanCvBody, ScanCvResponse};
use serde_json::Value as JsonValue;
use std::io;

#[async_std::main]
async fn main() -> std::io::Result<()> {
    OutMessages::Ready(String::from("waiting for credentials")).print();

    let mut buffer = String::new();
    let mut api = Api::new();

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

        handle_in_message(input, &mut api).await.print();
    }
}

async fn handle_in_message(input: InMessages, api: &mut Api) -> OutMessages {
    match input {
        InMessages::SetCredentials(credentials) => {
            if let Err(err) = api.set_credentials(credentials) {
                return OutMessages::ErrorAuth(err);
            }

            let health_req: Result<GetStatusResponse, String> = api.get("/api/v1/health").await;
            if let Err(err) = health_req {
                return OutMessages::ErrorAuth(err.to_string());
            }

            let key_info_res: Result<ApiKeyInfo, String> = api.get("/api/v1/auth/keyinfo").await;
            let key_info = match key_info_res {
                Err(err) => return OutMessages::ErrorAuth(err.to_string()),
                Ok(v) => v,
            };
            let mut has_scraper_role = false;
            for role in key_info.roles {
                if role.is_scraper() {
                    has_scraper_role = true;
                    break;
                }
            }
            if !has_scraper_role {
                let err_msg = "provided key does not have scraper role (nr 1)";
                return OutMessages::ErrorAuth(String::from(err_msg));
            }

            OutMessages::Ok
        }
        InMessages::GetSecret(args) => {
            let key = match &args.key {
                Some(v) => v.as_str(),
                None => return OutMessages::ErrorInvalidInput(String::from("key is required")),
            };
            match api.get_secret::<JsonValue>(&args.encryption_key, key).await {
                Ok(v) => OutMessages::Secret(v),
                Err(err) => OutMessages::ErrorApi(err),
            }
        }
        InMessages::GetUsersSecret(args) => {
            match api.get_users_secret(&args.encryption_key, args.key).await {
                Ok(v) => OutMessages::UsersSecret(v),
                Err(err) => OutMessages::ErrorApi(err),
            }
        }
        InMessages::GetUserSecret(args) => {
            match api.get_user_secret(&args.encryption_key, args.key).await {
                Ok(v) => OutMessages::UserSecret(v),
                Err(err) => OutMessages::ErrorApi(err),
            }
        }
        InMessages::SendCv(cv) => {
            let body = Some(ScanCvBody { cv });
            let res: Result<ScanCvResponse, String> =
                api.post("/api/v1/scraper/scanCV", body).await;

            match res {
                Err(err) => OutMessages::ErrorApi(err),
                Ok(_) => OutMessages::Ok,
            }
        }
        InMessages::Ping => OutMessages::Pong,
    }
}
