# Scale 15X Gossip Protocol in CDNs Demo app

this app is default listen public_interface:12356

after you run this app, you able to use `nc -U /tmp/12356` to get rpc console

you can join other node by using `/join $ip` in console

Predefined command include

- `/join $ip` # join other node
- `/list` # list all node
- `/say $something` # boardcast $something to all node and show to all console connection
- `/bash $cmd` # push bash $cmd to all node, this is high risk, please remove function if you want to play
- `ping $ip/all` # push `ping -c 1 $ip|grep from` to all server, if arg is all, then loop this command to all node

Use by your own risk

The libary this app use is by [serf](https://github.com/hashicorp/serf), if you design your app, I will prefer you check serf first, this app is not designed well, this just working code implement in short time.
