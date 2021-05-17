# Deployment

## Binary deployment

Download the corresponding version from the [Releases](https://github.com/indes/flowerss-bot/releases) page, unzip it and run it.

## Docker deployment

1. Download the configuration file
Create a new `config.yml` file in the project directory


```bash
mkdir ~/flowerss &&\
wget -O ~/flowerss/config.yml \
    https://raw.githubusercontent.com/indes/flowerss-bot/master/config.yml.sample
```


2. Modify the configuration file

```bash
vim ~/flowerss/config.yml
```

Modify the sqlite path in the configuration file (if sqlite is used as the database):
```yaml
sqlite:
  path: /root/.flowerss/data.db
```

3. Run

```shell script
docker run -d -v ~/flowerss:/root/.flowerss indes/flowerss-bot
```

## Source code compilation and deployment

```shell script
git clone https://github.com/indes/flowerss-bot && cd flowerss-bot
make build
./flowerss-bot
```



## Configuration

Create a new `config.yml` file based on the following template.

```yml
bot_token: XXX
#Multiple telegraph_token can be in array format:
# telegraph_token:
#-token_1
#-token_2
telegraph_token: xxxx
user_agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36
preview_text: 0
disable_web_page_preview: false
socks5: 127.0.0.1:1080
update_interval: 10
error_threshold: 100
telegram:
  endpoint: https://xxx.com/
mysql:
  host: 127.0.0.1
  port: 3306
  user: user
  password: pwd
  database: flowerss
sqlite:
  path: ./data.db
allowed_users:
  -123
  -234
```

Configuration instructions:

| Configuration item        | Meaning                                                      | Required or not     |
| --------------------------| ------------------------------------------------------------ | ------------------- | 
| bot_token                 | Telegram Bot Token                                           | Required            |
| telegraph_token           | Telegraph Token, used to transfer original text to Telegraph | Ignorable (do not transfer original text to Telegraph) |
| preview_text              | Plain text preview word count (without Telegraph)            | can be ignored (default 0, 0 is disabled) |
| user_agent                | User Agent                                                   |           Ignorable |
| disable_web_page_preview  | Whether to disable web page preview                          | Ignorable (default false, true to disable) |
| update_interval           | RSS feed scan interval (minutes)                             | Ignorable (default 10) |
| error_threshold           | Maximum number of source errors                              | Ignorable (default 100) |
| socks5                    | Used in environments where the Telegram API cannot work      | Ignorable (Can connect to the Telegram API server normally) |
| mysql | MySQL database configuration | Ignorable (using SQLite) |
| sqlite | SQLite configuration | Ignorable (this item is invalid when mysql is configured) |
| telegram.endpoint | Custom telegram bot api url | Ignorable (use the default api url) |
| allowed_users | Telegram id of users allowed to use bot, | can be ignored, all users can use bot when it is empty |
