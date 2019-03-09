+++ 
draft = true
date = 2019-03-09T14:28:03+01:00
title = "Setting up screen with irssi"
slug = "always-on-irc" 
tags = ["irc","linux","irssi","screen"]
categories = ["irc","linux"]
+++

So up till recently I ran my irc client (irssi) on my local system. I had no problem with this, but my continious joining
and leaving was causing some annoyance with some people. And I don't blame them.
The thing was that I packed up my laptop and as it jumped from network to network it was causing some spam.

So I ssh'd into my Raspberry Pi and started working on it. First we need to install the programs, which can be
done with `sudo apt install screen irssi`. Sure, it's not the most shiny up to date version of these programs, but it'll
do.
After this, I copied over my irssi config to the Pi, spawned a screen session with `sreen -dmS irc` and connected to it
with `screen -r irc`. Then with my config copied over I can start up irssi and bamm, you have a persistent irc
client running.
The benefit of doing this that I have backlog for when I'm not online and I create less joinspam.

This has been running fine, but I might update it to tmux in the future. In which the screen commands change to `tmux
new-session irc` and `tmux attach -t irc`.

# irssi plugins

I'll list some the plugins that I use below and the purpose they serve:

* [desktop-notify.pl](https://github.com/irssi/scripts/blob/master/scripts/desktop-notify.pl): Sends notification using the Desktop Notifications Specification
* [mouse.pl](https://github.com/irssi/scripts.irssi.org/blob/master/scripts/mouse.pl): control irssi using mouse clicks and gestures
* [recentdepart.pl](https://github.com/irssi/scripts/blob/master/scripts/recentdepart.pl): Filters quit/part/join/nick notices based on time since last message. (Similar to weechat's smartfilter).
* [scriptassist.pl](https://github.com/irssi/scripts/blob/master/scripts/scriptassist.pl): keeps your scripts on the cutting edge
* [tmux-nicklist-portable.pl](https://github.com/irssi/scripts/blob/master/scripts/tmux-nicklist-portable.pl): displays a list of nicks in a separate tmux pane	
* [usercount.pl](https://github.com/irssi/scripts/blob/master/scripts/usercount.pl): Adds a usercount for a channel as a statusbar item
* [trackbar.pl](https://github.com/irssi/scripts/blob/master/scripts/trackbar.pl): https://github.com/irssi/scripts/blob/master/scripts/trackbar.pl
* [smartfilter.pl](https://github.com/irssi/scripts/blob/master/scripts/smartfilter.pl): Improved smart filter for join, part, quit, nick messages
