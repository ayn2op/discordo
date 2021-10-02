# discordo &middot; [![build](https://github.com/ayntgl/discordo/actions/workflows/build.yml/badge.svg)](https://github.com/ayntgl/discordo/actions/workflows/build.yml) [![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/ayntgl/discordo/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/ayntgl/discordo)](https://goreportcard.com/report/github.com/ayntgl/discordo)

Discordo is a lightweight, secure, and feature-rich Discord terminal client.

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
  - Partial [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

### Building from source

```bash
git clone https://github.com/ayntgl/discordo
cd discordo
go build

# optional
sudo mv ./discordo /usr/local/bin
```

### Package managers

- Arch Linux (unofficial, AUR, may be outdated): `yay -S discordo-git`

## Usage

1. Run the `discordo` executable with no arguments. A new default configuration will be created at `~/.config/discordo/config.toml` on first startup.
2. Log in using the account email and password (first-time login) and click on the "Login" button to continue.

> By default, Discordo utilizes OS-specific keyring to store credentials such as client authentication token. However, if you prefer not to use a keyring (not recommended), you may set the `token` field in the configuration file and Discordo will prioritize the usage of the provided token to login instead of keyring.

### Configuration

Discordo aims to be highly configurable, it may be easily customized via a configuration file. It creates a default configuration file on the first start-up. The default and newly created configuration file is located at `$HOME/.config/discordo/config.toml`.

### Clipboard support

On Linux, clipboard support requires:

- `xclip` or `xsel` for X11.
- `wl-clipboard` for Wayland.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
