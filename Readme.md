# README.md for Discord Whitelist Bot

## Overview

This Discord Whitelist Bot is designed to manage a user whitelist, utilizing GitHub as a backend for storing and retrieving whitelist data. It allows server administrators to add or remove users from the whitelist via Discord commands. This document outlines the setup and usage of the bot.

## Prerequisites

- A Discord account and a server where you have administrative privileges.
- A GitHub account with a repository for storing the whitelist.
- Go programming environment.

## Setup

1. **Clone the Repository:**

   - Clone the bot's code repository to your server or local machine.

2. **Configure Environment Variables:**

   - Locate the `.env.example` file in the cloned directory.
   - Copy this file and rename the copy to `.env`.
   - Fill in the values in `.env` with your credentials:
     - `DISCORD_TOKEN`: Your Discord Bot token.
     - `GITHUB_TOKEN`: Your GitHub Personal Access Token.
     - `GITHUB_USERNAME`: Your GitHub username.
     - `GITHUB_URL`: URL of the GitHub repository used for the whitelist.
   - Ensure your environment correctly loads variables from the `.env` file.

3. **Deploy the Bot:**
   - After setting up the `.env` file, run the bot using `go run`.

## Usage

1. **Invite the Bot to Your Server:**

   - Use the invite link generated in the Discord Developer Portal.

2. **Using Commands:**

   - **Add to Whitelist:** Use `/whitelist add` to add a user.
   - **Remove from Whitelist:** Use `/whitelist remove` to remove a user.

## Troubleshooting

- Ensure the `.env` file is correctly set up and located in the root directory of the project.
- Check if the environment variables are properly loaded when the application starts.
- Check Discord Bot permissions on your server.
- Verify GitHub Token permissions.

## Support

For support, please create an issue in the GitHub repository linked in `GITHUB_URL` or contact the bot developer.

---

This document provides a basic overview of setting up and using the Discord Whitelist Bot. Ensure you follow each step carefully for successful deployment and operation.
