# # Joystick.TV Example Bot in GO

This is an example "Magic Word" bot that integrates with [JoystickTV](https://joystick.tv)'s Chatbot API.

**NOTE**: You must be over the age of 18 to create an account.

This bot will send a message to the streamer's chat when someone says the "magic word". After it's been installed by the streamer, just type that word or phrase in their chat, and you'll see the message.

## Requirements

* JoystickTV account
* API Key
* API Secret
* Access to the API

For questions, join Joystick.TV's [Discord](https://discord.gg/zKvCf8hrGP) server,
and ask in their #support channel

## Setup

Create a `.env` file, and fill with the appropriate values

```
JOYSTICKTV_CLIENT_ID=changeme
JOYSTICKTV_CLIENT_SECRET=changeme
JOYSTICKTV_API_HOST=changeme
JOYSTICKTV_HOST=changeme
```

Run `go get` to install dependencies.

## Configuring

The example "magic word" is `tacos`. You can change this in `main.go`.

## Running

`go run .`