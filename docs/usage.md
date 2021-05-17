## Usage

command:

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

### Channel subscription usage

1. Add Bot as Channel Administrator
2. Send related commands to Bot

Commands supported by Channel subscription:

```
/sub @ChannelID [url] Subscribe
/unsub @ChannelID [url] Unsubscribe
/list @ChannelID View current subscription
/check @ChannelID Check current subscription
/unsuball @ChannelID Cancel all subscriptions
/activeall @ChannelID open all subscriptions
/setfeedtag @ChannelID [sub id] [tag1] [tag2] Set subscription tags (set up to three tags, separated by spaces)
/import Import OPML files
/export @ChannelID Export OPML file
/pauseall @ChannelID Pause all subscriptions
```

**ChannelID is only available if it is set to Public Channel. If it is a Private Channel, you can temporarily set it to Public, and change it to Private after the subscription is completed, which does not affect Bot push messages. **

For example, to subscribe to the t.me/debug channel [Ruan Yifeng's weblog](http://www.ruanyifeng.com/blog/atom.xml) RSS updates:

1. Add Bot to the debug channel manager list
2. Send the `/sub @debug http://www.ruanyifeng.com/blog/atom.xml` command to the Bot
