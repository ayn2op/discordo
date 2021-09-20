# discordo &middot; [![build](https://github.com/ayntgl/discordo/actions/workflows/build.yml/badge.svg)](https://github.com/ayntgl/discordo/actions/workflows/build.yml) [![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/ayntgl/discordo/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/ayntgl/discordo)](https://goreportcard.com/report/github.com/ayntgl/discordo)

[WIP] Discordo is a lightweight, secure, and feature-rich Discord terminal client. It is highly configurable and has a minimalistic user interface.

![Preview](.github/preview.png)

## Features

- Lightweight
- Secure
- Configurable
- Cross-platform
- Minimalistic
- Feature-rich
    - Mouse & clipboard support
    - 2-Factor authentication
    - Desktop notifications
    - [WIP] Discord-flavored markdown

## Installation

### Building from source

```bash
git clone https://github.com/rigormorrtiss/discordo
cd discordo
go build

# optional
sudo mv ./discordo /usr/local/bin
```

### Package managers

- Arch Linux (AUR, may be outdated): `paru -S discordo-git`

## Usage

- Run the executable in a new Terminal window.

By default, Discordo utilizes OS-specific keyring to store credentials such as client authentication token. However, if you prefer not to use a keyring, you may set the `token` field in the configuration file (`~/.config/discordo/config.json`) and Discordo will prioritize the usage of `token` field to login instead of keyring. 

- Log in using the email and password (first-time login) and click on the "Login" button to continue.

### Default keybindings

Global:

- `Alt` + `1`: Sets the focus on the guilds TreeView.
- `Alt` + `2`: Sets the focus on the messages TextView.
- `Alt` + `3`: Sets the focus on the message InputField.

Messages:

- `k` or `Up`: Selects the message just before the currently selected message.
- `j` or `Down`: Selects the message just after the currently selected message.
- `g` or `Home`: Selects the first message rendered.
- `G` or `End`: Selects the last message rendered.

- `r`: Reply to the selected message without mention.
- `R`: Reply to the currently selected message with mention.

Input:

- `Esc`: Unselect the currently selected message. This is useful for canceling an inline reply message.

### Clipboard support

On Linux, clipboard support requires:

- `xclip` or `xsel` for X11.
- `wl-clipboard` for Wayland.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
