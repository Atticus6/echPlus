mod vless;

use axum::{
    extract::{
        ws::{Message, WebSocket},
        State, WebSocketUpgrade,
    },
    http::StatusCode,
    response::{IntoResponse, Response},
    routing::get,
    Router,
};
use clap::Parser;
use std::{net::SocketAddr, sync::Arc};
use tokio::net::TcpStream;
use tracing::{error, info, warn};
use uuid::Uuid;

#[derive(Parser, Debug, Clone)]
#[command(author, version, about = "VLESS WebSocket Server", long_about = None)]
struct Args {
    #[arg(short, long, env = "UUID", default_value = "147258369-1234-5678-9abc-def012345678")]
    uuid: String,

    #[arg(short, long, env = "PORT", default_value = "3325")]
    port: u16,
}

#[derive(Clone)]
struct AppState {
    uuid: Uuid,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info".into()),
        )
        .init();

    let args = Args::parse();
    let uuid = Uuid::parse_str(&args.uuid)?;

    let state = AppState { uuid };

    let app = Router::new()
        .route("/", get(ws_handler))
        .route("/health", get(health_handler))
        .with_state(Arc::new(state));

    let addr = SocketAddr::from(([0, 0, 0, 0], args.port));
    info!("VLESS Server listening on {}", addr);
    info!("UUID: {}", uuid);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

async fn health_handler() -> impl IntoResponse {
    (StatusCode::OK, "OK")
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<Arc<AppState>>,
) -> Response {
    ws.on_upgrade(move |socket| handle_socket(socket, state))
}

async fn handle_socket(socket: WebSocket, state: Arc<AppState>) {
    if let Err(e) = handle_vless_session(socket, state).await {
        error!("Session error: {}", e);
    }
}

async fn handle_vless_session(mut ws: WebSocket, state: Arc<AppState>) -> anyhow::Result<()> {
    use futures_util::StreamExt;

    // Read VLESS request header
    let header_data = match ws.recv().await {
        Some(Ok(Message::Binary(data))) => data,
        Some(Ok(_)) => anyhow::bail!("Expected binary message"),
        Some(Err(e)) => anyhow::bail!("WebSocket error: {}", e),
        None => anyhow::bail!("Connection closed"),
    };

    // Parse VLESS request
    let (target_addr, command, payload) = vless::parse_request(&header_data, state.uuid)?;

    if command != vless::CMD_TCP {
        anyhow::bail!("Unsupported command: {}", command);
    }

    info!("Connecting to {}", target_addr);

    // Connect to target
    let remote = TcpStream::connect(&target_addr).await?;

    info!("Connected to {}", target_addr);

    // Send VLESS response
    let response = vless::build_response();
    ws.send(Message::Binary(response)).await?;

    // Send initial payload if exists
    if !payload.is_empty() {
        use tokio::io::AsyncWriteExt;
        let (mut remote_read, mut remote_write) = remote.into_split();
        remote_write.write_all(&payload).await?;
        
        // Bidirectional relay
        let (ws_sender, ws_receiver) = ws.split();

        let ws_to_remote = relay_ws_to_tcp(ws_receiver, remote_write);
        let remote_to_ws = relay_tcp_to_ws(remote_read, ws_sender);

        tokio::select! {
            r1 = ws_to_remote => {
                if let Err(e) = r1 {
                    warn!("WS->Remote error: {}", e);
                }
            }
            r2 = remote_to_ws => {
                if let Err(e) = r2 {
                    warn!("Remote->WS error: {}", e);
                }
            }
        }
    } else {
        // Bidirectional relay
        let (ws_sender, ws_receiver) = ws.split();
        let (remote_read, remote_write) = remote.into_split();

        let ws_to_remote = relay_ws_to_tcp(ws_receiver, remote_write);
        let remote_to_ws = relay_tcp_to_ws(remote_read, ws_sender);

        tokio::select! {
            r1 = ws_to_remote => {
                if let Err(e) = r1 {
                    warn!("WS->Remote error: {}", e);
                }
            }
            r2 = remote_to_ws => {
                if let Err(e) = r2 {
                    warn!("Remote->WS error: {}", e);
                }
            }
        }
    }

    info!("Session ended: {}", target_addr);
    Ok(())
}

async fn relay_ws_to_tcp(
    mut ws_rx: futures_util::stream::SplitStream<WebSocket>,
    mut tcp_tx: tokio::net::tcp::OwnedWriteHalf,
) -> anyhow::Result<()> {
    use futures_util::StreamExt;
    use tokio::io::AsyncWriteExt;

    while let Some(msg) = ws_rx.next().await {
        match msg? {
            Message::Binary(data) => {
                tcp_tx.write_all(&data).await?;
            }
            Message::Close(_) => break,
            _ => {}
        }
    }
    Ok(())
}

async fn relay_tcp_to_ws(
    mut tcp_rx: tokio::net::tcp::OwnedReadHalf,
    mut ws_tx: futures_util::stream::SplitSink<WebSocket, Message>,
) -> anyhow::Result<()> {
    use futures_util::SinkExt;
    use tokio::io::AsyncReadExt;

    let mut buf = vec![0u8; 32 * 1024];
    loop {
        let n = tcp_rx.read(&mut buf).await?;
        if n == 0 {
            break;
        }
        ws_tx.send(Message::Binary(buf[..n].to_vec())).await?;
    }
    Ok(())
}
