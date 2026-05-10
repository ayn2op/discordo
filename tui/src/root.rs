use config::Config;
use crossterm::event::{Event, KeyCode};
use keyring_core::Entry;
use ratatui::{Frame, layout::Rect, widgets::Block};
use tea::{Cmd, Model as _, Sub};

use crate::{chat, login};

const TOKEN_ENV_VAR_KEY: &str = "DISCORDO_TOKEN";
const KEYRING_SERVICE: &str = "discordo";
const KEYRING_USER: &str = "token";

enum Inner {
    Login(login::Model),
    Chat(chat::Model),
}

impl Inner {
    fn init() -> (Self, Cmd<Msg>) {
        let (model, cmd) = login::Model::init();
        (Inner::Login(model), cmd.map(Msg::Login))
    }

    fn update(&mut self, msg: Msg) -> Cmd<Msg> {
        match (self, msg) {
            (Inner::Login(m), Msg::Login(msg)) => m.update(msg).map(Msg::Login),
            (Inner::Login(m), Msg::Crossterm(e)) => {
                m.update(login::Msg::Crossterm(e)).map(Msg::Login)
            }
            (Inner::Chat(m), Msg::Chat(msg)) => m.update(msg).map(Msg::Chat),
            (Inner::Chat(m), Msg::Crossterm(e)) => {
                m.update(chat::Msg::Crossterm(e)).map(Msg::Chat)
            }
            _ => Cmd::None,
        }
    }

    fn view(&self, frame: &mut Frame, area: Rect) {
        match self {
            Inner::Login(m) => m.view(frame, area),
            Inner::Chat(m) => m.view(frame, area),
        }
    }
}

pub enum Msg {
    Crossterm(Event),
    Login(login::Msg),
    Chat(chat::Msg),
    TokenFromEnvVar(Option<String>),
    TokenFromKeyring(Option<String>),
}

pub struct Model {
    config: Config,
    inner: Option<Inner>,
}

impl tea::Model for Model {
    type Msg = Msg;

    fn init() -> (Self, Cmd<Self::Msg>) {
        let config = Config::load().unwrap();
        (Model { config, inner: None }, Cmd::task(get_token_from_env()))
    }

    fn update(&mut self, msg: Self::Msg) -> Cmd<Self::Msg> {
        match msg {
            Msg::Crossterm(Event::Key(key)) if key.code == KeyCode::Esc => Cmd::Quit,
            Msg::TokenFromEnvVar(Some(_)) => {
                let (model, cmd) = chat::Model::init();
                self.inner = Some(Inner::Chat(model));
                cmd.map(Msg::Chat)
            }
            Msg::TokenFromEnvVar(None) => Cmd::task(get_token_from_keyring()),
            Msg::TokenFromKeyring(Some(_)) => todo!(),
            Msg::TokenFromKeyring(None) => {
                let (inner, cmd) = Inner::init();
                self.inner = Some(inner);
                cmd
            }
            msg => self.inner.as_mut().map_or(Cmd::None, |inner| inner.update(msg)),
        }
    }

    fn view(&self, frame: &mut Frame, area: Rect) {
        match &self.inner {
            Some(inner) => inner.view(frame, area),
            None => frame.render_widget(Block::bordered().title("LOADING"), area),
        }
    }

    fn subscriptions(&self) -> Sub<Self::Msg> {
        Sub::Event(|event| Some(Msg::Crossterm(event)))
    }
}

async fn get_token_from_env() -> Msg {
    Msg::TokenFromEnvVar(std::env::var(TOKEN_ENV_VAR_KEY).ok())
}

async fn get_token_from_keyring() -> Msg {
    let _ = keyring::use_native_store(true);
    Msg::TokenFromKeyring(
        Entry::new(KEYRING_SERVICE, KEYRING_USER)
            .and_then(|entry| entry.get_password())
            .ok(),
    )
}
