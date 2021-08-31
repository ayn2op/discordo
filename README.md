# discordo &middot; [![build](https://github.com/rigormorrtiss/discordo/actions/workflows/build.yml/badge.svg)](https://github.com/rigormorrtiss/discordo/actions/workflows/build.yml) [![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/rigormorrtiss/discordo/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/rigormorrtiss/discordo)](https://goreportcard.com/report/github.com/rigormorrtiss/discordo)

Discordo is a lightweight, secure, and feature-rich Discord terminal client.

![Preview](.github/preview.png)

## Features

- Lightweight & secure
- Easy-to-use & cross-platform
- Configurable & minimalistic
- Feature-rich
    - Discord-flavored markdown
    - Clipboard support
    - 2-Factor Authentication

## Installation

### Building

```bash
git clone https://github.com/rigormorrtiss/discordo
cd discordo && go build

# optional
sudo mv ./discordo /usr/local/bin
```

## Package managers

```bash
# (AUR) Arch Linux - development version (may be outdated)
yay -S discordo-git
```

## Getting Started

- Run the executable in a new Terminal window.

By default, Discordo utilizes OS-specific keyring to store credentials such as client authentication token. However, if you prefer not to use a keyring, you may set the `token` field in the configuration file (`~/.config/discordo/config.json`) and Discordo will prioritize the usage of `token` field to login instead of keyring. 

- Log in using the email and password (first-time login) and click on the "Login" button to continue.

### Default Keybindings

Global:

- `Alt` + `1`: Sets the focus on the guilds TreeView.
- `Alt` + `2`: Sets the focus on the messages TextView.
- `Alt` + `3`: Sets the focus on the message InputField.

TextView:

- `Alt` + `k` or `Alt` + `Up`: Selects the message just before the currently selected message.
- `Alt` + `j` or `Alt` + `Down`: Selects the message just after the currently selected message.
- `Alt` + `g` or `Alt` + `Home`: Selects the first message rendered in the TextView.
- `Alt` + `G` or `Alt` + `End`: Selects the last message rendered in the TextView.

### Clipboard

- Requires `xclip` or `xsel` for X11.
- Requires `wl-clipboard` for Wayland.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
