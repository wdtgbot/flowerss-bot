

### There are a lot of prompts similar to `Create telegraph page error: FLOOD_WAIT_7` in the log.

The reason is that the request to create the Telegraph page too quickly triggered the interface restriction. You can try to add multiple Telegraph tokens to the configuration file.


### How to apply for Telegraph Token?

If you want to use the in-app instant preview, you must fill in the `telegraph_token` configuration item in the configuration file. The Telegraph Token application command is as follows:
```bash
curl https://api.telegra.ph/createAccount?short_name=flowerss&author_name=flowerss&author_url=https://github.com/indes/flowerss-bot
```

The value of the access_token field in the returned JSON is the Telegraph Token.


### How to get my telegram id?
You can refer to this page to get: https://botostore.com/c/getmyid_bot/
