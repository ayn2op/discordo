# Discordo

**Discordo** is a lightweight, secure, and feature-rich Discord terminal client.  

## Warning

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.

## Authentification

There are two ways to login:  
> In both cases, the authentication token is stored securely in the default OS-specific keyring.

**Username / Password login**

1. Run `discordo` without arguments.  
2. Enter your email and password then click on the "Login" button to continue.

**Token login**

Use the **`--token`** flag:  
```
discordo --token "OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"
```

## Keybindings

Keybindings are configurable in the [configuration file](#configuration).

| Action                | Keybinding |
| --------------------- | ---------- |
| **Guilds Tree**       |            |
| Focus                 | Alt + g    |
|                       |            |
| **Messages Text**     |            |
| Focus                 | Alt + m    |
| Show image            | i          |
| Copy message content  | c          |
| Reply without mention | r          |
| Reply with mention    | R          |
| Select reply          | s          |
| Reply previous        | Up arrow   |
| Select next           | Down arrow |
| Select first          | Home       |
| Select last           | End        |
|                       |            |
| **Message Input**     |            |
| Focus                 | Alt + i    |
| Send message          | Enter      |
| Paste from clipboard  | Ctrl + v   |
| Launch editor         | Ctrl + e   |

## Configuration

The configuration file is stored in the following location:

| Operating System | Configuration File Location                             |
| ---------------- | ------------------------------------------------------- |
| Unix             | `$HOME/.config/discordo/config.yml`                     |
| Darwin           | `$HOME/Library/Application Support/discordo/config.yml` |
| Windows          | `%AppData%/discordo/config.yml`                         |

From there, you can edit keybindings or change the theme.

## Log files

The log file is stored in the following location:

| Operating System | Log File Location                        |
| ---------------- | ---------------------------------------- |
| Unix             | `$HOME/.cache/discordo/logs.txt`         |
| Darwin           | `$HOME/Library/Caches/discordo/logs.txt` |
| Windows          | `%LocalAppData%/discordo/logs.txt`       |
