# discordo &middot; [![ci](https://img.shields.io/github/workflow/status/ayntgl/discordo/ci?color=5865F2&logo=github)](https://github.com/ayntgl/discordo/actions/workflows/ci.yml) [![license](https://img.shields.io/github/license/ayntgl/discordo?color=5865F2&logo=github)](https://github.com/ayntgl/discordo/blob/master/LICENSE) [![discord](https://img.shields.io/discord/903923641819992064?color=5865F2&logo=discord&logoColor=white)](https://discord.gg/u49vegcfnh)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

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
make build

# optional
sudo mv ./discordo /usr/local/bin
```

### Package managers

- Arch Linux (AUR, unofficial): `yay -S discordo-git`

## Usage

1. Run the `discordo` executable with no arguments.

> On first startup, a default configuration file will be created at `$HOME/.config/discordo.toml` on Unix, `%AppData%/discordo.toml` on Windows, and `$HOME/Library/Application Support/discordo.toml` on Darwin.

2. Log in using the account email and password (first-time login) and click on the "Login" button to continue.

> Note: by default, Discordo utilizes OS-specific keyring to store credentials such as client authentication token. However, if you prefer not to use a keyring (not recommended), you may set the `token` field in the configuration file and Discordo will prioritize the usage of the provided token to login instead of keyring.

### Clipboard support

On Linux, clipboard support requires:

- `xclip` or `xsel` for X11.
  - Ubuntu: `apt install xclip`
  - Arch Linux: `pacman -S xclip`
  - Fedora: `dnf install xclip`
- `wl-clipboard` for Wayland.
  - Ubuntu: `apt install wl-clipboard`
  - Arch Linux: `pacman -S wl-clipboard`
  - Fedora: `dnf install wl-clipboard`

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
