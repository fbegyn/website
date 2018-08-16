+++ 
draft = false
date = 2018-08-16T12:58:17+02:00
title = "Making debian packages on Arch Linux"
slug = "" 
tags = ["Arch","Debian","Linux","Packages"]
categories = ["linux"]
+++

Lately I've been helping with a little APT repo, and it's been fun. The thing I struggled a bit with was making a `.deb` package on my system ("Hey, did you know I use Arch Linux?"). Not because this is badly documented, but a lot of the documentation is actually making use of `devscripts`, `dh_make` or other helpers, which for some unkown reason all refuse to work on my system. TBD what the cause of this is.

So I started doing things manually, and what follows are my notes on the process.  
**DISCLAIMER**: I'm relatively sure people that know this better then me would yell at me for some of the things that I'm about to write. So this is by no means a guide, this is so I won't forget how I did things.

# Directory structure

Generally the layout of the directory is can be straigh forward. I'll use the example of my `prometheus` package.
```
prometheus-2.3.2
├── DEBIAN
│   ├── compat
│   ├── control
│   ├── postinst
│   ├── postrm
│   └── rules
├── etc
│   └── prometheus
│       └── prometheus.yml
├── lib
│   └── systemd
│       └── system
│           └── prometheus.service
├── usr
│   └── bin
│       ├── prometheus
│       └── promtool
└── var
    └── lib
            └── prometheus
```

Ignore the `DEBIAN` folder at the moment. All the other directories should remind you of some familiar structure, and indeed it does! This represents the were the files are going to go relatively to `/`. As you can see binaries, config files, ... all end up in the places where they belong.

## DEBIAN folder

The debian folder has all the things that are related to how the package gets installed, the meta data of the package and all the other wonderfull things that come with packaging.

### control

Has all the meta data of the package. A minimal control file is mentioned below (all the field in there are required).
```
Package: <package name>
Version: <package version>
Maintainer: <maintainer>
Architecture: <any, i386, amd64, ...> 
Description: <description>
```

Most of it is pretty straight forward. Just the `Architecture` is something to pay attention to. And the `Description` can be extended with multi-line just by identing the next lines with a space.

### rules, postinst, postrm

All these are scripts, and it's important to remember that scripts have to be executable. The `rules` specify how the package is actually made. `postinst` and `postrm` are scripts that are executed after installation and removal of the packages. These are mostly used for file permissions.

```
DEBIAN/rules
------------
#!/usr/bin/make -f
# See debhelper(7) (uncomment to enable)
# output every command that modifies files on the build system.
#export DH_VERBOSE = 1


# see FEATURE AREAS in dpkg-buildflags(1)
#export DEB_BUILD_MAINT_OPTIONS = hardening=+all

# see ENVIRONMENT in dpkg-buildflags(1)
# package maintainers to append CFLAGS
#export DEB_CFLAGS_MAINT_APPEND  = -Wall -pedantic
# package maintainers to append LDFLAGS
#export DEB_LDFLAGS_MAINT_APPEND = -Wl,--as-needed


%:
        dh $@ --with-systemd


# dh_make generated override targets
# This is example for Cmake (See https://bugs.debian.org/641051 )
#override_dh_auto_configure:
#       dh_auto_configure -- #  -DCMAKE_LIBRARY_PATH=$(DEB_HOST_MULTIARCH)
```
