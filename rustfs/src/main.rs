use axum::{
    extract::{Multipart, Path, State},
    http::{header, StatusCode},
    response::{IntoResponse, Response},
    routing::{delete, get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{env, path::PathBuf, sync::Arc};
use tokio::{fs, io::AsyncWriteExt};
use tower_http::cors::CorsLayer;
use tracing::{error, info};
use uuid::Uuid;

#[derive(Clone)]
struct AppState {
    data_dir: PathBuf,
}

#[derive(Serialize, Deserialize)]
struct FileInfo {
    id: String,
    filename: String,
    size: u64,
    content_hash: String,
    path: String,
}

#[derive(Serialize)]
struct UploadResponse {
    success: bool,
    file: FileInfo,
}

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Get data directory from environment
    let data_dir = env::var("RUSTFS_DATA_DIR").unwrap_or_else(|_| "/data".to_string());
    let port = env::var("RUSTFS_PORT").unwrap_or_else(|_| "9000".to_string());

    let data_path = PathBuf::from(data_dir);

    // Create data directory if it doesn't exist
    fs::create_dir_all(&data_path)
        .await
        .expect("Failed to create data directory");

    info!("RustFS starting on port {}", port);
    info!("Data directory: {:?}", data_path);

    let state = Arc::new(AppState {
        data_dir: data_path,
    });

    let app = Router::new()
        .route("/health", get(health_check))
        .route("/upload", post(upload_file))
        .route("/files/:id", get(download_file).delete(delete_file))
        .route("/files/:id/info", get(file_info))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let addr = format!("0.0.0.0:{}", port);
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind to address");

    info!("RustFS listening on {}", addr);

    axum::serve(listener, app).await.expect("Server error");
}

async fn health_check() -> impl IntoResponse {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "rustfs"
    }))
}

async fn upload_file(
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Result<Json<UploadResponse>, StatusCode> {
    let mut filename = String::new();
    let mut file_data = Vec::new();

    while let Some(field) = multipart.next_field().await.map_err(|_| StatusCode::BAD_REQUEST)? {
        let name = field.name().unwrap_or("").to_string();

        if name == "file" {
            filename = field
                .file_name()
                .unwrap_or("unnamed")
                .to_string();
            file_data = field.bytes().await.map_err(|_| StatusCode::BAD_REQUEST)?.to_vec();
        }
    }

    if file_data.is_empty() {
        return Err(StatusCode::BAD_REQUEST);
    }

    // Generate file ID and hash
    let file_id = Uuid::new_v4().to_string();
    let mut hasher = Sha256::new();
    hasher.update(&file_data);
    let content_hash = hex::encode(hasher.finalize());

    // Create directory structure: data/{first_two_chars}/{file_id}
    let prefix = &file_id[..2];
    let dir_path = state.data_dir.join(prefix);
    fs::create_dir_all(&dir_path)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let file_path = dir_path.join(&file_id);

    // Write file
    let mut file = fs::File::create(&file_path)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    file.write_all(&file_data)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    info!("File uploaded: {} ({})", filename, file_id);

    Ok(Json(UploadResponse {
        success: true,
        file: FileInfo {
            id: file_id.clone(),
            filename,
            size: file_data.len() as u64,
            content_hash,
            path: format!("/files/{}", file_id),
        },
    }))
}

async fn download_file(
    State(state): State<Arc<AppState>>,
    Path(id): Path<String>,
) -> Result<Response, StatusCode> {
    let prefix = &id[..2];
    let file_path = state.data_dir.join(prefix).join(&id);

    if !file_path.exists() {
        return Err(StatusCode::NOT_FOUND);
    }

    let data = fs::read(&file_path).await.map_err(|e| {
        error!("Error reading file: {}", e);
        StatusCode::INTERNAL_SERVER_ERROR
    })?;

    Ok(([(header::CONTENT_TYPE, "application/octet-stream")], data).into_response())
}

async fn delete_file(
    State(state): State<Arc<AppState>>,
    Path(id): Path<String>,
) -> Result<StatusCode, StatusCode> {
    let prefix = &id[..2];
    let file_path = state.data_dir.join(prefix).join(&id);

    if !file_path.exists() {
        return Err(StatusCode::NOT_FOUND);
    }

    fs::remove_file(&file_path).await.map_err(|e| {
        error!("Error deleting file: {}", e);
        StatusCode::INTERNAL_SERVER_ERROR
    })?;

    info!("File deleted: {}", id);

    Ok(StatusCode::NO_CONTENT)
}

async fn file_info(
    State(state): State<Arc<AppState>>,
    Path(id): Path<String>,
) -> Result<Json<FileInfo>, StatusCode> {
    let prefix = &id[..2];
    let file_path = state.data_dir.join(prefix).join(&id);

    if !file_path.exists() {
        return Err(StatusCode::NOT_FOUND);
    }

    let metadata = fs::metadata(&file_path)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let data = fs::read(&file_path)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let mut hasher = Sha256::new();
    hasher.update(&data);
    let content_hash = hex::encode(hasher.finalize());

    Ok(Json(FileInfo {
        id: id.clone(),
        filename: id.clone(),
        size: metadata.len(),
        content_hash,
        path: format!("/files/{}", id),
    }))
}
