# flowerss bot

[![Build Status](https://github.com/indes/flowerss-bot/workflows/Release/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow %3ARelease)
[![Test Status](https://github.com/indes/flowerss-bot/workflows/Test/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow %3ATest)
![Build Docker Image](https://github.com/indes/flowerss-bot/workflows/Build%20Docker%20Image/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/indes/flowerss-bot)](https://goreportcard.com/report/github.com/indes/flowerss-bot)
![GitHub](https://img.shields.io/github/license/indes/flowerss-bot.svg)

[Installation and Use Document](https://flowerss-bot.now.sh/)

<img src="https://github.com/rssflow/img/raw/master/images/rssflow_demo.gif" width = "300"/>

## Features

-Common RSS Bot functions
-Support instant view in Telegram app
-Support for subscribing to RSS news for Group and Channel
-Rich subscription settings

## Installation and use

For detailed installation and usage, please refer to the project [Usage Document](https://flowerss-bot.now.sh/).

Use the command:

```
/sub [url] Subscribe (url is optional)
/unsub [url] Unsubscribe (url is optional)
/list View current subscriptions
/set Set subscription
/check check current subscription
/setfeedtag [sub id] [tag1] [tag2] Set the subscription tag (set up to three tags, separated by spaces)
/setinterval [interval] [sub id] Set subscription refresh frequency (multiple sub ids can be set, separated by spaces)
/activeall open all subscriptions
/pauseall Pause all subscriptions
/import Import OPML files
/export Export OPML file
/unsuball cancel all subscriptions
/help help
```
For detailed usage, please refer to the project [Usage Document](https://flowerss-bot.now.sh/#/usage).
