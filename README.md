# D‑C2 🛰️

Discord‑based command‑and‑control (C2) client for educational and defensive research purposes.

> This project is provided **only** for lawful use, such as learning, red‑teaming in environments you own, or testing your own defenses.  
> **Do not** run it on systems or networks you do not own or explicitly control.

## Overview 🔍

The client connects to a Discord bot and opens a dedicated text channel per device inside your Discord server.  
Channels are grouped by OS:

- `Windows` category for Windows clients
- `Linux` category for Linux clients
- `Mac` category for macOS clients
- `Other` for everything else

Each device gets its own channel named like:

```text
hostname-username-192-168-1-10
```

The first message in that channel includes a readable session label:

```text
hostname | username | 192.168.1.10
```

From that channel you control the client using emoji‑based commands.

## Features / Actions ⚙️

From the per‑device channel, the client supports:

- `🏃‍♂️ <command>` – run an OS command  
  - Windows: executed via `cmd.exe`
  - Linux/macOS: executed via `/bin/bash -c`
- `📸` – capture screenshots of all active displays and upload them
- `👇 <path>` – download a file from the client (size < 8 MB)
- `☝️ <path> *attach` – upload an attached file from Discord to the given path on the client
- `💀` – terminate the client process

All command output is sent back to the channel. Long output is written to a temp file and uploaded instead of being pasted directly.

## Project Structure 📁

```text
discord-c2-server/
├── client.go       # Entry point, session setup, per-device channel wiring
├── env.go          # Environment loading and global configuration (token, guild ID)
├── categories.go   # Discord category and device channel helpers
├── handler.go      # Discord message handler and emoji command implementation
├── go.mod
├── go.sum
└── README.md
```

## Requirements 📦

- Go 1.19 or newer
- A Discord account
- A Discord application with a bot user and bot token
- A Discord server (guild) where the bot has permission to:
  - Create categories
  - Create text channels
  - Read and send messages
  - Upload files

## Configuration 🔑

Configuration is done via a `.env` file in the project root.

Create `.env`:

```env
DISCORD_BOT_TOKEN=YOUR_BOT_TOKEN_HERE
DISCORD_GUILD_ID=YOUR_GUILD_ID_HERE
```

- `DISCORD_BOT_TOKEN` – the bot token from the Discord Developer Portal (without the `Bot ` prefix)
- `DISCORD_GUILD_ID` – the numeric ID of the Discord server where you want channels created

> Do **not** commit `.env` to public repositories. Keep your token private and rotate it if it ever leaks.

## Building 🏗️

From the project root:

```bash
go build -o discord-c2-client client.go
```

Cross‑compile examples:

- Linux 64‑bit (from any OS):

  ```bash
  GOOS=linux GOARCH=amd64 go build -o client-linux client.go
  ```

- Windows 64‑bit (from any OS):

  ```bash
  GOOS=windows GOARCH=amd64 go build -o client-windows.exe client.go
  ```

## Running the client 🚀

1. Ensure `.env` is present with a valid bot token and guild ID.
2. Invite the bot to your Discord server with the required permissions.
3. Run the compiled binary on the target machine, for example:

   ```bash
   ./client-linux
   ```

   or on Windows:

   ```powershell
   .\client-windows.exe
   ```

4. In your Discord server:
   - A category for the OS (`Windows`, `Linux`, `Mac`, or `Other`) is created if it doesn’t exist.
   - A text channel for that device appears inside the category.
   - The first message in that channel shows:
     - IP address
     - Username
     - Hostname
     - OS
     - Current working directory

5. Use the emoji commands in that channel to interact with the client.

## Legal / Ethical Use ⚖️

- Only run this client on systems and networks you own or are explicitly authorized to test.
- Do not use it for unauthorized access, surveillance, or any activity that violates laws, terms of service, or organizational policies.
- The author(s) and contributors are not responsible for any misuse or damage caused by this software.

By using this project, you agree to abide by all applicable laws and use it responsibly.
