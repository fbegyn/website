---
title: Finding a problem in Go's rtnetlink (or how to nerdsnipe me)
date: 2021-03-07
tags: [ linux, networking, netlink ]
draft: false
---

# Finding a problem in Go's rtnetlink (or how to nerdsnipe me)

A couple of weeks ago a friend of mine asked me to look into a little project of
his, since something didn't seem to work and I had some prior experience with the
netlink library. So I took a look at his program and encountered the following error:

```shell
Error:      	Received unexpected error:
                netlink receive: invalid argument
```

## Diving in ...

OK, so something is going wrong somewhere. After reviewing [the
library](https://github.com/jsimonetti/rtnetlink) and it documentation, nothing
seemed to indicate a wrong use of the library. So I did some experiments with
different shapes of the
[LinkXDP](https://github.com/jsimonetti/rtnetlink/blob/master/link.go#L536)
struct of the library to see if I couldn't coax some shape to work. Below is a
snippet to show the layout of the struct.

```go
// LinkXDP holds Express Data Path specific information
type LinkXDP struct {
	FD         uint32
	ExpectedFD uint32
	Attached   uint8
	Flags      uint32
	ProgID     uint32
}
```

As I was running experiments with different combinations of inputs of the
`LinkXDP` type, every iteration ended up with the same error.

```shell
Error:      	Received unexpected error:
                netlink receive: invalid argument
```

This didn't seem right, so I decided to dive a bit deeper. I ran the code with
`-exec 'sudo strace'` to dive deeper into what the system was sending. Which lead
me to the following output for the netlink message related to the XDP program
being loaded.

```shell
[
    {{nla_len=6, nla_type=IFLA_UNSPEC}, "\x00\x00"},
    {{nla_len=5, nla_type=IFLA_IFNAME}, ""},
    {{nla_len=8, nla_type=IFLA_LINK}, 0},
    {{nla_len=5, nla_type=IFLA_QDISC}, ""},
    {{nla_len=44, nla_type=IFLA_XDP}, [
        {{nla_len=8, nla_type=IFLA_XDP_FD}, 6},
        {{nla_len=8, nla_type=IFLA_XDP_EXPECTED_FD}, 0},
        {{nla_len=5, nla_type=IFLA_XDP_ATTACHED}, XDP_ATTACHED_NONE},
        {{nla_len=8, nla_type=IFLA_XDP_FLAGS}, XDP_FLAGS_SKB_MODE},
        {{nla_len=8, nla_type=IFLA_XDP_PROG_ID}, 0}
    ]}
]
```

Strange, this all seems to match with what we are sending. Let's do some more
research on [this part of the rtnetlink
API](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/rtnetlink.c#L2911).

## ... into the deep end

Reading through [the
code](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/rtnetlink.c#L2911)
I discovered the following thing. The API does not use the `IFLA_XDP_PROG_ID` and
`IFLA_XDP_ATTACHED` when receiving message related to link XDP. In fact, when it
sees these fields, [it errors out of the
operation](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/rtnetlink.c#L2894).

```c
if (xdp[IFLA_XDP_ATTACHED] || xdp[IFLA_XDP_PROG_ID]) {
    err = -EINVAL;
	goto errout;
}
```

This explains why no combination worked for the library, since it always
[encoded](https://github.com/jsimonetti/rtnetlink/blob/master/link.go#L563) these fields.

```go
func (xdp *LinkXDP) encode(ae *netlink.AttributeEncoder) error {
	ae.Uint32(unix.IFLA_XDP_FD, xdp.FD)
	ae.Uint32(unix.IFLA_XDP_EXPECTED_FD, xdp.ExpectedFD)
	ae.Uint8(unix.IFLA_XDP_ATTACHED, xdp.Attached)
	ae.Uint32(unix.IFLA_XDP_FLAGS, xdp.Flags)                // <- encoding FLAGS
	ae.Uint32(unix.IFLA_XDP_PROG_ID, xdp.ProgID)             // <- encodding PROG_ID
	return nil
}
```

The solution here is straight forward, these fields should not be encoded since
the kernel will throw an `EINVAL`. Seems like these fields are read only! While
looking through the source code, I also stumbled upon the following snippet:

```c
static const struct nla_policy ifla_xdp_policy[IFLA_XDP_MAX + 1] = {
	[IFLA_XDP_UNSPEC]	= { .strict_start_type = IFLA_XDP_EXPECTED_FD },
	[IFLA_XDP_FD]		= { .type = NLA_S32 },
	[IFLA_XDP_EXPECTED_FD]	= { .type = NLA_S32 },
	[IFLA_XDP_ATTACHED]	= { .type = NLA_U8 },
	[IFLA_XDP_FLAGS]	= { .type = NLA_U32 },
	[IFLA_XDP_PROG_ID]	= { .type = NLA_U32 },
};
```

Here, my eye fell on the types for the `FD` and `EXPECTED_FD`, `NLA_S32`. Signed
integers, while we encode `uint32` in the library. That seems strange, so some
more reading lead me to [the following
snippet](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/dev.c#L9328):

```c
if (fd >= 0) {
	new_prog = bpf_prog_get_type_dev(fd, BPF_PROG_TYPE_XDP,
					 mode != XDP_MODE_SKB);
	if (IS_ERR(new_prog))
		return PTR_ERR(new_prog);
}

if (expected_fd >= 0) {
	old_prog = bpf_prog_get_type_dev(expected_fd, BPF_PROG_TYPE_XDP,
					 mode != XDP_MODE_SKB);
	if (IS_ERR(old_prog)) {
		err = PTR_ERR(old_prog);
		old_prog = NULL;
		goto err_out;
	}
}
```

So when the passed integer is negative, we don't load the BPF program,
essentially clearing it. That seems like a handy feature to have, so lets
implement it so that the library encodes it as `int32`.

## On the road to a solution

First, lets just not encode those `ATTACHED` and `PROG_ID` fields. As mentioned,
this is an easy solution. While we're at it, lets also change the types of the
`FD` fields to `int32` and encode them correctly.

```go
// LinkXDP holds Express Data Path specific information
type LinkXDP struct {
	FD         int32
	ExpectedFD int32
	Attached   uint8
	Flags      uint32
	ProgID     uint32
}

...

func (xdp *LinkXDP) encode(ae *netlink.AttributeEncoder) error {
	ae.Int32(unix.IFLA_XDP_FD, xdp.FD)
	ae.Int32(unix.IFLA_XDP_EXPECTED_FD,  xdp.ExpectedFD)
	ae.Uint32(unix.IFLA_XDP_FLAGS, xdp.Flags)
	// XDP_ATtACHED and XDP_PROG_ID are things that only can return from the kernel,
	// not be send, so we don't encode them.
	// source: https://elixir.bootlin.com/linux/v5.10.15/source/net/core/rtnetlink.c#L2894
	// ae.Uint8(unix.IFLA_XDP_ATTACHED, xdp.Attached)
	// ae.Uint32(unix.IFLA_XDP_PROG_ID, xdp.ProgID)
	return nil
}
```

Add some tests to validate the new behavior and tests some edge cases and voila!
Things work ... except that `ae.Int32` does not exist yet (as of v1.4.0 of the
[netlink
library](https://github.com/mdlayher/netlink/blob/master/CHANGELOG.md#v140) they
do), but that's nothing that a [quick
PR](https://github.com/mdlayher/netlink/pull/185) can't solve. With these in
place, we can finally run tests and make a
[PR](https://github.com/jsimonetti/rtnetlink/pull/113) to the rtnetlink library
to fix XDP program loading.

After our changes, running the original code from the start with `-exec 'sudo
strace'` gives the following:

```shell
[
    {{nla_len=6, nla_type=IFLA_UNSPEC}, "\x00\x00"},
    {{nla_len=5, nla_type=IFLA_IFNAME}, ""},
    {{nla_len=8, nla_type=IFLA_LINK}, 0},
    {{nla_len=5, nla_type=IFLA_QDISC}, ""},
    {{nla_len=28, nla_type=IFLA_XDP}, [
        {{nla_len=8, nla_type=IFLA_XDP_FD}, 6},
        {{nla_len=8, nla_type=IFLA_XDP_EXPECTED_FD}, 0},
        {{nla_len=8, nla_type=IFLA_XDP_FLAGS}, XDP_FLAGS_SKB_MODE}
    ]}
]
```

Which is what we (and the kernel) expected.

## Almost there

PR submitted, waiting for CI and ... failure? Strange, the tests succeed local.
After banging my head on my desk for a while, I decided to try it on another
machine. The tests also failed there, the only difference? The linux kernel
version.

As it turns out, the `REPLACE` function is only supported as of kernel version
`5.7` ([5.6](https://elixir.bootlin.com/linux/v5.6/source/net/core/dev.c#L8662)
and [5.7](https://elixir.bootlin.com/linux/v5.7/source/net/core/dev.c#L8674)).
So, we need a check in place to validate that the kernel supports the feature we
want to check.

```go
// getKernelVersion gets the kernel version through syscall.uname
func getKernelVersion() (kernel, major, minor int, err error) {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return 0, 0, 0, err
	}

	end := bytes.IndexByte(uname.Release[:], 0)
	versionStr := uname.Release[:end]

	if count, _ := fmt.Sscanf(string(versionStr), "%d.%d.%d", &kernel, &major, &minor); count < 2 {
		err = fmt.Errorf("failed to parse kernel version from: %q", string(versionStr))
	}
	return
}

// kernelMinReq checks if the runtime kernel is sufficient
// for the test
func kernelMinReq(t *testing.T, kernel, major int) {
	k, m, _, err := getKernelVersion()
	if err != nil {
		t.Fatalf("failed to get host kernel version: %v", err)
	}
	if k < kernel || k == kernel && m < major {
		t.Skipf("host kernel (%d.%d) does not meet test's minimum required version: (%d.%d)",
			k, m, kernel, major)
	}
}
```

Using the `kernelMinReq` function at the start of a test will skip the test if
the host kernel does not match the minimal required version.

## Sources

* [rtnetlink library](https://github.com/jsimonetti/rtnetlink)
* [netlink library](https://github.com/mdlayher/netlink)
* [Linux kernel code](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/rtnetlink.c)
* [Linux kernel code](https://elixir.bootlin.com/linux/v5.10.15/source/net/core/dev.c)
