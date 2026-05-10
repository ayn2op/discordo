use crossterm::event::{Event, EventStream};
use futures::{FutureExt, StreamExt, future::BoxFuture, stream::BoxStream};
use ratatui::{Frame, layout::Rect};
use tokio::{sync::mpsc, task};

pub enum Cmd<Msg> {
    None,
    Quit,
    Msg(Msg),
    Task(BoxFuture<'static, Msg>),
    Batch(Vec<Cmd<Msg>>),
}

impl<Msg: Send + 'static> Cmd<Msg> {
    pub fn task(future: impl Future<Output = Msg> + Send + 'static) -> Self {
        Self::Task(Box::pin(future))
    }

    pub fn batch(cmds: impl IntoIterator<Item = Self>) -> Self {
        Self::Batch(cmds.into_iter().collect())
    }

    pub fn map<Next: Send + 'static>(
        self,
        f: impl FnOnce(Msg) -> Next + Clone + Send + 'static,
    ) -> Cmd<Next> {
        match self {
            Self::None => Cmd::None,
            Self::Quit => Cmd::Quit,
            Self::Msg(msg) => Cmd::Msg(f(msg)),
            Self::Task(future) => Cmd::task(future.map(f)),
            Self::Batch(cmds) => {
                Cmd::Batch(cmds.into_iter().map(|cmd| cmd.map(f.clone())).collect())
            }
        }
    }
}

pub enum Sub<Msg> {
    None,
    Event(fn(Event) -> Option<Msg>),
    Stream(BoxStream<'static, Msg>),
    Batch(Vec<Sub<Msg>>),
}

impl<Msg: Send + 'static> Sub<Msg> {
    pub fn batch(subs: impl IntoIterator<Item = Self>) -> Self {
        Self::Batch(subs.into_iter().collect())
    }
}

pub trait Model: Sized {
    type Msg: Send + 'static;
    fn init() -> (Self, Cmd<Self::Msg>);
    fn update(&mut self, msg: Self::Msg) -> Cmd<Self::Msg>;
    fn view(&self, frame: &mut Frame, area: Rect);
    fn subscriptions(&self) -> Sub<Self::Msg> {
        Sub::None
    }
}

pub async fn run<M: Model>() {
    let mut terminal = ratatui::init();

    let (tx, mut rx) = mpsc::unbounded_channel::<M::Msg>();

    let (mut model, cmd) = M::init();
    if handle_cmd(cmd, tx.clone()) {
        ratatui::restore();
        return;
    }
    handle_sub(model.subscriptions(), tx.clone());

    _ = terminal.draw(|frame| model.view(frame, frame.area()));

    while let Some(msg) = rx.recv().await {
        let cmd = model.update(msg);
        if handle_cmd(cmd, tx.clone()) {
            break;
        }

        _ = terminal.draw(|frame| model.view(frame, frame.area()));
    }
    ratatui::restore();
}

fn handle_cmd<Msg: Send + 'static>(cmd: Cmd<Msg>, tx: mpsc::UnboundedSender<Msg>) -> bool {
    match cmd {
        Cmd::None => false,
        Cmd::Quit => true,
        Cmd::Msg(msg) => tx.send(msg).is_err(),
        Cmd::Task(future) => {
            task::spawn(future.map(move |msg| tx.send(msg)));
            false
        }

        Cmd::Batch(cmds) => {
            for cmd in cmds {
                if handle_cmd(cmd, tx.clone()) {
                    return true;
                }
            }
            false
        }
    }
}

fn handle_sub<Msg: Send + 'static>(sub: Sub<Msg>, tx: mpsc::UnboundedSender<Msg>) {
    match sub {
        Sub::None => (),
        Sub::Stream(mut stream) => {
            task::spawn(async move {
                while let Some(msg) = stream.next().await {
                    if tx.send(msg).is_err() {
                        break;
                    }
                }
            });
        }
        Sub::Event(mapper) => {
            task::spawn(async move {
                let mut events = EventStream::new();
                while let Some(Ok(event)) = events.next().await {
                    if let Some(msg) = mapper(event)
                        && tx.send(msg).is_err()
                    {
                        break;
                    }
                }
            });
        }
        Sub::Batch(subs) => {
            for sub in subs {
                handle_sub(sub, tx.clone());
            }
        }
    }
}

// pub fn add(left: u64, right: u64) -> u64 {
//     left + right
// }

// #[cfg(test)]
// mod tests {
//     use super::*;

//     #[test]
//     fn it_works() {
//         let result = add(2, 2);
//         assert_eq!(result, 4);
//     }
// }
