---
title: Linux TC - part 01
date: 2021-01-01
tags: [ linux, restic, nixos, backup]
draft: true
---

# Setting up restic backups on Nixos

Inspired by [this
blogpost](https://christine.website/blog/borg-backup-2021-01-09), I decided to
set up my own backup solution of my server data with
[Restic](https://restic.net/). I currently use it for backup of my workstations,
but i still do it the old fashioned way: manually. Luckily, Nixos has
`servives.restic.backups` available, so let's dive in.

## Starting with Restic: repositories

Restic works with "repositories" too which you can send backups. So first, we
need to create some repository. In this post, I'll set up a sync to my Google
drive through [rclone](). This is a bit more complex [than some of the other
options](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#),
but I use the Google drive for plenty of things and it'll most likely always be accessible.

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

```
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

```
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

```
extraBackupArgs = [ "--exclude-file=/home/backups/important/data/not-important" ];
```

and plenty more things are available, see `nixos-option services.restic.backups.gdrive`.

There is one option worthwhile expanding on in this post: `timerConfig`. This
option makes use of a systemd timer to enable schedueled backups. The syntac for
it can be found in `man systemd.timer` and `man systemd.time`. Below is an
example to take backups, every Saturday at 23:00.

```
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

```
# The naming follows restic-backups-<backup name>, so for us here it would be

sudo systemctl start restic-backups-gdrive.service
```
