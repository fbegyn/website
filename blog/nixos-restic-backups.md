---
title: Setting up restic backups on Nixos
date: 2021-01-12
tags: [ linux, restic, nixos, backup]
draft: false
---

# Setting up restic backups on Nixos

Inspired by [this
blogpost](https://christine.website/blog/borg-backup-2021-01-09), I decided to
set up my own backup solution of my server data with
[Restic](https://restic.net/). I currently use it for backup of my workstations,
but i still do it the old fashioned way: manually. Luckily, Nixos has
`services.restic.backups` available, so let's dive in.

## Starting with Restic: repositories

Restic works with "repositories" to which you can send backups. So first, we need
to create some repository. In this post, I'll set up a sync to my Google drive
through [rclone](https://rclone.org/). This is a bit more complex [than some of
the other
options](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#),
but I use the Google drive for plenty of things and it'll most likely always be
accessible.

So, first we need to setup `rclone` to talk to Google drive. Thankfully, `rclone`
has an amazing setup guide, just type `rclone config` and follow the steps of the
setup guide. I setup my Google drive with the name `gdrive`.

Now, I need to setup a `restic` repository.

```
restic -r rclone:gdrive:/backups init
enter password for new repository:
```

Keep this password safe, losing it will lock you out of the backups. The flipside
of having encrypted backups. Off course, we could also let Nixos create this
repository:

```nix
...
services.restic.backups = {
  gdrive = {
    user = "backups";
    repository = "rclone:gdrive:/backups";
    initialize = true; # initializes the repo, don't set if you want manual control
    passwordFile = "<path>";
  };
};
```

Which will setup the repository with the password provided by the `passwordFile` keyword.

## Configuring Nixos to backup with Restic

We now got a repository ready, time to get some data in it!

```nix
services.restic.backups = {
  gdrive = {
    ...
    paths = [ "/home/backups/important/data" ];
    user = "backups";
  };
};

```

This will backup some important data in the home folder of the `backups` user. If
there are directories that should be excluded or you want to use some other
options for `restic` that you want to user, you can set that with the
`extraBackupArgs` option:

```nix
extraBackupArgs = [ "--exclude-file=/home/backups/important/data/not-important" ];
```

and plenty more things are available, see `nixos-option services.restic.backups.gdrive`.

There is one option worthwhile expanding on in this post: `timerConfig`. This
option makes use of a systemd timer to enable schedueled backups. The syntac for
it can be found in `man systemd.timer` and `man systemd.time`. Below is an
example to take backups, every Saturday at 23:00.

```nix
services.restic.backups = {
  gdrive = {
    ...
    timerConfig = {
      onCalendar = "saturday 23:00";
    };
  };
};
```

All that is left to say is that you can now trigger backups manually by using the
generated systemd service:

```shell
# The naming follows restic-backups-<backup name>, so for us here it would be

$ sudo systemctl start restic-backups-gdrive.service
```

## Restoring, the most important part

Restic offers multiple ways to restore your backups. You can restore a snapshot
to a specific location with a `--target <id>` parameter.

```shell
$ restic -r rclone:gdrive:/backups restore <id> --target <path>`
```

Which will restore the snapshot to the `<path>`. You can modify what will be
restored by using a `--host <host>` (restoring a snapshot from a specific host)
and `--path <path>` (to only restore a snapshot of a specific path). There are
also of course the `--include/--exclude <sub>` which will include/exclude parts
of the snapshot (these are by default case sensitive, can be made case
insensitive by prefixing `i`).

Aside from that, you can mount snapshot like a file system (so you can copy from
it as usual). This is easily done with the `mount <mountpoint>` command. This will mount the
repository on the mountpoint specified.

```shell
$ ls -al /tmp/backups/
total 48
dr-xr-xr-x  1 francis francis     0 11 jan 23:37 ./
drwxrwxrwt 36 root    root    45056 11 jan 23:37 ../
dr-xr-xr-x  1 francis francis     0 11 jan 23:37 hosts/
dr-xr-xr-x  1 francis francis     0 11 jan 23:37 ids/
dr-xr-xr-x  1 francis francis     0 11 jan 23:37 snapshots/
dr-xr-xr-x  1 francis francis     0 11 jan 23:37 tags/

~
$ ls -al /tmp/backups/snapshots/
total 0
dr-xr-xr-x 1 francis francis 0 11 jan 23:37 ./
dr-xr-xr-x 1 francis francis 0 11 jan 23:37 ../
dr-xr-xr-x 3 francis francis 0 11 jan 20:52 2021-01-11T20:52:25+01:00/
dr-xr-xr-x 3 francis francis 0 11 jan 21:43 2021-01-11T21:43:54+01:00/
lrwxrwxrwx 1 francis francis 0 11 jan 21:43 latest -> 2021-01-11T21:43:54+01:00/
```

You can browse through the backups by various means: tags, hosts, id, snapshot
dates, ... .

And last but not least, you can just dump a file from the snapshot to stdout with
the `dump` command. This handy when you just need that 1 SQL dump to restore it
for example.

## Sources

* [restic docs](https://restic.readthedocs.io/en/stable/index.html)
* [restic nixos options](https://search.nixos.org/options?channel=20.09&from=0&size=30&sort=relevance&query=restic)
* [rclone docs](https://rclone.org/docs/)
