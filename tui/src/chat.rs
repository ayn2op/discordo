use crossterm::event::Event;
use ratatui::{Frame, layout::Rect, widgets::Block};
use tea::Cmd;

pub enum Msg {
    Crossterm(Event),
}

pub struct Model;

impl tea::Model for Model {
    type Msg = Msg;

    fn init() -> (Self, Cmd<Self::Msg>) {
        (Self, Cmd::None)
    }

    fn update(&mut self, msg: Self::Msg) -> Cmd<Self::Msg> {
        match msg {
            Msg::Crossterm(_event) => Cmd::None,
        }
    }

    fn view(&self, frame: &mut Frame, area: Rect) {
        frame.render_widget(Block::bordered().title("CHAT"), area);
    }
}
