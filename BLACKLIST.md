# Blacklist info

The black list is a newline delimited file of wallet addresses. It can also support comments with the `#` character.

## Default Location

By default, the nordicenergy binary looks for the file `./.ngy/blaklist.txt`.

## Example File
```
net1spshr72utf6rwxseaz339j09ed8p6f8ke370zj
net1uyshu2jgv8w465yc8kkny36thlt2wvel89tcmg  # This is a comment
net1r4zyyjqrulf935a479sgqlpa78kz7zlcg2jfen

```

## Details

Each transaction added to the tx-pool has its `to` and `from` address checked against this blacklist. 
If there is a hit, the transaction is considered invalid and is dropped from the tx-pool.