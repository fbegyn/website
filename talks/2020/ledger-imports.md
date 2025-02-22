---
title: hledger imports for beginners
date: 2020-10-05
author: Francis Begyn
email: francis@begyn.be
institute: Zeus WPI
site: https://francis.begyn.be
draft: false
---

# Plain text accounting

---

* small recap: what is plain text accounting?
	* plain text file
	* double-entry accounting
	* tooling to create reports

---

## Automating the import

* Manual imports offer a lot of flexibility
	* they also take a lot of time
	* handling backlogs becomes slow
	* especially with a lot of repetative transactions

* luckily, `(h)ledger` has to option to import from csv files
	* fast import of backlog

* ledger: `convert`
* hledger: `import`

---

## CSV files (or why I dislike Belgian banks)

* csv need some processing (most of the time)
	* sources use different (non-standard) formats
	* some work in cents, others in euros
	* time formats are incorrect or can't be parsed by `hledger`
	* ...

* 1 transaction per row
* all transaction must have the same amount of columns
* must contain an amount and date
	* other fields can be set in the rules file or read from the csv file as well

---

## hledger rule file

* rule files decribe to hledger how to read the csv file
* rule files are tied to the structure of the csv
* some basic keywords:
	* `skip n`: skips the first `n` lines
	* `fields ...`: comma separate field identifier (for classifying columns in the csv)
	* `currency`: can be set in the file, are identified from the csv file
	* `accountx`: account #`x` that takes part in the transaction 
	* `amountx`: amount of funds transfered matching the `x` account
	* `description`: description of the transaction
	* `comment`: comment that will be added to the transaction

---

## hledger rule file

```
skip 1
fields date, description, amount

# specify the date field's format 
# - not needed here since date is Y/M/D
# date-format %-d/%-m/%Y
# date-format %-m/%-d/%Y
# date-format %Y-%h-%d

currency $
account1 assets:bank:checking

if (TO|FROM) SAVINGS
  account2 assets:bank:savings
```

---

## example: Zeus tab

* The processed csv file
	* `amount` is divided by 100 (cents -> euros)

```
"amount","creditor","debtor","id","issuer", ...
20,"thecy","Zeus",3933,"iepoev","GELD via b ...
0.75,"Zeus","thecy",3992,"Tap","1 Ice Tea", ...
0.7,"Zeus","thecy",4026,"Tap","1 Cola","201 ...
0.75,"Zeus","thecy",4033,"Tap","1 Ice Tea", ...
0.75,"Zeus","thecy",4057,"Tap","1 Ice Tea", ...
2.35,"Zeus","thecy",4062,"Tap","1 Cola and  ...
6,"Zeus","thecy",4150,"thecy","Chinees 18/1 ...
0.75,"Zeus","thecy",4152,"Tap","1 Ice Tea", ...
0.75,"Zeus","thecy",4153,"Tap","1 Ice Tea", ...
30,"thecy","Zeus",4162,"iepoev","GELD banco ...
2.04,"basho","thecy",4174,"thecy","Desparad ...
```

---

## example: Zeus tab

```
skip 1
fields am, creditor, debitor,,, msg, date
date-format %FT%T.000%Ez
# equivalent of date-format
# %Y-%m-%dT%H:%M:%S.000+02:00
account1 assets:be:zeus:tab
account2 expenses:voedsel
description %debitor to %creditor: %msg
amount1 -%am EUR
amount2 %am EUR

if %creditor thecy
	amount1 %am EUR
	account2 income:zeus:tab
	amount2 -%am EUR
```

---

# Demo time!
