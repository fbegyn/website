---
date: 2020-12-02
title: "AoC 2020: day 1 and 2"
tags: [aoc,programming]
draft: false
---
# AoC 2020: day 1 and 2

It's that glorious time of the year again -what? no, not the holidays-, the
advent of code has started again -yes, it's holidays related-. The time of the
year where my biorithm gets a good shaking, I finally wake up at time and I
frustrute myself over tiny typos that cost me precious hours on the leaderboard.

For those that don't know it: Advent of code is an online programming challenge
that poses a programming challenge every day (at 00:00 UTC+5) -that's 06:00
UTC-1, my time- up to December 25th. At that time thousands of programming
enthousiasts across the world flock to [the site](https://adventofcode.com/) to
solve the problems as fast as possible to claim a price on the global
leaderboard. And in the hours following that, many more will solve the problem
for their private leaderboards that they have between friends, colleagues,
comrades, enemies, ...

Last year [I somewhat stopped halfway through](https://github.com/fbegyn/AoC2019)
:(, so lets try it again [this year](https://github.com/fbegyn/aoc2020).

## Day 1

Ah, it seems we're not saving Christmas this year, we're going on a well deserved
holiday. Appearently our holiday destination uses some weird currency called
"stars". Hmm, anyway we'll figure out how to get the money as we go.

What's that? I need to fill in my expenses? Urgh, fine. The accountant lets me
know that I need to go through the expenses of this year and give him the
multiplication of the `2` expenses that sum together to `2020` -how this helps
any accountant, I have no idea-.

I quickly read in my expenses into [a tiny
program](https://github.com/fbegyn/aoc2020/blob/main/go/cmd/day01/main.go) and
skim through them to see which ones add up to 2020.

```go
func part1(expenses []int) (int, error) {
	for i, v := range expenses {
		for _, w := range expenses[i+1:] {
			if v+w == 2020 {
				return v * w, nil
			}
		}
	}
	return 0, errors.New("no solution found for part 2")
}
```

I give the  report back to the account,  talk a bit about my  planned holiday and
this dammed  curreny, when  one of the  accounts mentions that  he still  has `2`
"stars" and gives  one to me for the  help. He also mention that I  can the other
one if I help him find which `3`  entries sum to `2020` -again, why?-. Not one to
let a challenge lay  around, I look back at the program to  see if I can't expand
it.

```go
func part2(expenses []int) (int, error) {
	t := helpers.MinInt(expenses)
	for i, v := range expenses {
		for j, w := range expenses[i+1:] {
			for _, z := range expenses[j+1:] {
				if v+w+z == 2020 {
					return v * w * z, nil
				}
			}
		}
	}
	return 0, errors.New("no solution found for part 2")
}
```

Which comes to a quick solution al in all. I happily take the second "star" and
start packing my bags. A friend walks by and mention something about the previous
challenge "If a combination reaches `2020` in part 1, then surely some
combinations can be skipped in part 2?". I try to ignore it, but if something is
worth doing, it's worth overdoing. I take another look at the code I had written
before and insert 3 line:

```go
func part2(expenses []int) (int, error) {
	t := helpers.MinInt(expenses)
	for i, v := range expenses {
		for j, w := range expenses[i+1:] {
			if v+w > 2020 - t {
				continue
			}
			for _, z := range expenses[j+1:] {
				if v+w+z == 2020 {
					return v * w * z, nil
				}
			}
		}
	}
	return 0, errors.New("no solution found for part 2")
}
```

If the first `2` elements sum to `2020`, there is no way the third can add to
`2020` again (since the elements are non-zero).

## Day 2

Walking up to my trusty [toboggan
supplier](https://en.wikipedia.org/wiki/Toboggan), I notice him shouting at his
computer. Something messed up the passwords and now he can't login. There are
still some random passwords and policies there. He asks if I can take a look at
the computer.

```
1-3 a: abcde
1-3 b: cdefg
2-9 c: ccccccccc
```

He mentions that everything before the `:` is a policy, and then behind the `:`
there are passswords. the number indicate the lower and upper limit that the
letter is allowed to occur in the password. Each policy matches to the password it shares
a line with. Looking at this, I parse the policy into their respective elements:

```go
type PasswordRule struct {
	limits []int
	letter string
}

func PasswordRuleFromLine(line string) (PasswordRule, string) {
	set1 := strings.Split(line, ":")
	set2 := strings.Split(set1[0], " ")
	set3 := strings.Split(set2[0], "-")

	lower, err := strconv.Atoi(set3[0])
	if err != nil {
		log.Fatalln(err)
	}
	upper, err := strconv.Atoi(set3[1])
	if err != nil {
		log.Fatalln(err)
	}

	return PasswordRule{
		limits: []int{lower, upper},
		letter: set2[1],
	}, set1[1]
}
```

and write some small code to validate the password for a rule:

```go
func (r *PasswordRule) ValidSled(p string) bool {
	count := strings.Count(p, r.letter)
	if count < r.limits[0] || r.limits[1] < count {
		return false
	}
	return true
}
```

A few moments later I can tell them the amount of valid password, when suddenly
he remembers that the definition of the policies was incorrect. Instead of an
indication the upper and lower limit of the letter occurances, the policies
indicate on which locations the letter can appear exclusively (so the letter can
only occur on `1` of the `2` locations).

```go
func (r *PasswordRule) ValidToboggan(p string) bool {
	count := 0
	for _, v := range r.limits {
		if p[v-1] == r.letter[0] {
			count += 1
		}
	}
	if count != 1 {
		return false
	}
	return true
}
```

## notes

* It's insane, I never get out of bed early, but this competition gets me up
  super early.
  
* Shoutout to my friend for inspiring me for the day 1 "speedup". Gotta go fast.

* Not sure if I'll keep up this writing style for the rest of the advent of code.
It's my blog, I do what I want here.
