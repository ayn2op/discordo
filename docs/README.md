# Discordo

Discordo is a lightweight, secure, and feature-rich Discord terminal client.

## Table of Contents

- [FAQ](./faq.md)
- [Configuration](./configuration.md)

## Warning

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.

## Authentication

There are two ways to login:

> In both cases, the authentication token is stored securely in the default OS-specific keyring.

### Email login

1. Run `discordo` without arguments.
2. Enter your email and password then click on the "Login" button to continue.

### Token login

Use the `--token` flag:

```
discordo --token "OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"
```

## Logs

The log file is created on first start-up at the following location:

| Operating System | Log File Location                        |
| ---------------- | ---------------------------------------- |
| Unix             | `$HOME/.cache/discordo/logs.txt`         |
| Darwin           | `$HOME/Library/Caches/discordo/logs.txt` |
| Windows          | `%LocalAppData%/discordo/logs.txt`       |
