use crossterm::event::{Event, KeyCode};
use ratatui::{Frame, layout::{Constraint, Layout, Rect}};
use ratatui_textarea::TextArea;
use strum::FromRepr;
use tea::Cmd;

#[derive(Default, Clone, Copy, FromRepr)]
enum Focused {
    #[default]
    Email,
    Password,
}

impl Focused {
    fn previous(self) -> Self {
        Self::from_repr((self as usize).saturating_sub(1)).unwrap_or(self)
    }

    fn next(self) -> Self {
        Self::from_repr(self as usize + 1).unwrap_or(self)
    }
}

pub enum Msg {
    Crossterm(Event),
}

pub struct Model {
    email_input: TextArea<'static>,
    password_input: TextArea<'static>,
    focused: Focused,
}

impl tea::Model for Model {
    type Msg = Msg;

    fn init() -> (Self, Cmd<Self::Msg>) {
        let mut email_input = TextArea::default();
        email_input.set_placeholder_text("Email");

        let mut password_input = TextArea::default();
        password_input.set_placeholder_text("Password");
        password_input.set_mask_char('*');

        (Self { email_input, password_input, focused: Default::default() }, Cmd::None)
    }

    fn update(&mut self, msg: Self::Msg) -> Cmd<Self::Msg> {
        match msg {
            Msg::Crossterm(event) => match event {
                Event::Key(key) if key.code == KeyCode::Enter => Cmd::None,
                Event::Key(key) if key.code == KeyCode::BackTab => {
                    self.focused = self.focused.previous();
                    Cmd::None
                }
                Event::Key(key) if key.code == KeyCode::Tab => {
                    self.focused = self.focused.next();
                    Cmd::None
                }
                e @ (Event::Key(_) | Event::Mouse(_)) => {
                    match self.focused {
                        Focused::Email => { self.email_input.input(e); }
                        Focused::Password => { self.password_input.input(e); }
                    }
                    Cmd::None
                }
                _ => Cmd::None,
            },
        }
    }

    fn view(&self, frame: &mut Frame, area: Rect) {
        let [email_area, password_area] =
            Layout::vertical([Constraint::Min(1), Constraint::Min(0)]).areas(area);
        frame.render_widget(&self.email_input, email_area);
        frame.render_widget(&self.password_input, password_area);
    }
}
