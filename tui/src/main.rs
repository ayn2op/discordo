mod chat;
mod login;
mod root;

#[tokio::main]
async fn main() {
    tea::run::<root::Model>().await;
}
