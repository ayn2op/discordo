# discordo &middot; [![build](https://github.com/rigormorrtiss/discordo/actions/workflows/build.yml/badge.svg)](https://github.com/rigormorrtiss/discordo/actions/workflows/build.yml)

Discordo is a terminal-based Discord client that aims to be lightweight, secure, and feature-rich.

![Preview](.github/preview.png)

## Features

- Lightweight
- Easy-to-use
- Secure
- Cross-platform
- Configurable
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

- Run the built executable in a new Terminal window.

By default, Discordo utilizes OS-specific keyring to store credentials such as client authentication token. However, if you prefer not to use a keyring, you may set the `DISCORDO_TOKEN` environment variable and Discordo will prioritize the environment variable and use it to login instead of keyring. 

- Log in using the email and password and click on the "Login" button to continue.

### Default Keybindings

- `Alt` + `1`: Sets the focus on the guilds list.
- `Alt` + `2`: Sets the focus on the channels treeview.
- `Alt` + `3`: Sets the focus on the messages textview.
- `Alt` + `4`: Sets the focus on the message inputfield.

### Clipboard

- Requires `xclip` or `xsel` for X11.
- Requires `wl-clipboard` for Wayland.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
