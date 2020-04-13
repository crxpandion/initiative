# Initiative
Initiative is a simple HTML UI that rolls and tracks initiative for DND style encounters.

## How to install
```
go get -u github.com/crxpandion/initiative
go install ./...
```
## How to use 
```
$ initiative -h
Usage of initiative:
  -b string
        (default "localhost:8080")
  -d string
        (default "./test")
```
the `-d` directory should contain a `players` folder and a `monsters` folder.

the csv files are of the format
```
<name of player/monster>,<initiative bonus (aka dex modifier)>
```
edits to either the player file or the monsters file will automatically update the server state, but the encounter page will need to reload. This also updates the turn order rolls currently so only do this between encounters
## TODO
* Somehow model statuses
* allow additions to the players/monsters mid encounter (such that it doesnt reroll the order)
