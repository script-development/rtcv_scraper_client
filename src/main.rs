pub mod api;
pub mod messages;
pub mod rtcv_types;

use api::Api;
use messages::{InMessages, OkContent, OutMessages};
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
                OutMessages::Error(err.to_string()).print();
                continue;
            }
            Ok(v) => v,
        };

        match handle_in_message(input, &mut api).await {
            Ok(v) => v.print(),
            Err(err) => OutMessages::Error(err).print(),
        };
    }
}

async fn handle_in_message(input: InMessages, api: &mut Api) -> Result<OutMessages, String> {
    match input {
        InMessages::SetCredentials(credentials) => {
            api.set_credentials(credentials)?;

            api.get::<GetStatusResponse>("/api/v1/health").await?;

            let key_info: ApiKeyInfo = api.get("/api/v1/auth/keyinfo").await?;
            let mut has_scraper_role = false;
            for role in key_info.roles {
                if role.is_scraper() {
                    has_scraper_role = true;
                    break;
                }
            }
            if !has_scraper_role {
                return Err(String::from(
                    "provided key does not have scraper role (nr 1)",
                ));
            }

            Ok(OutMessages::Ok(OkContent::Empty))
        }

        InMessages::GetSecret(args) => {
            let key = match &args.key {
                Some(v) => v.as_str(),
                None => return Err(String::from("key is required")),
            };

            api.get_secret::<JsonValue>(&args.encryption_key, key)
                .await
                .map(|v| OutMessages::Ok(OkContent::Secret(v)))
        }

        InMessages::GetUsersSecret(args) => api
            .get_users_secret(&args.encryption_key, args.key)
            .await
            .map(|v| OutMessages::Ok(OkContent::UsersSecret(v))),

        InMessages::GetUserSecret(args) => api
            .get_user_secret(&args.encryption_key, args.key)
            .await
            .map(|v| OutMessages::Ok(OkContent::UserSecret(v))),

        InMessages::SendCv(cv) => {
            let body = Some(ScanCvBody { cv });
            api.post::<ScanCvResponse, _>("/api/v1/scraper/scanCV", body)
                .await?;

            Ok(OutMessages::Ok(OkContent::Empty))
        }

        InMessages::Ping => Ok(OutMessages::Pong),
    }
}
