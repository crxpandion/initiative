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
  -m string
         (default "./test/monsters.csv,./test/monsters_2.csv")
  -p string
         (default "./test/players.csv")
```
the csv files are of the format
```
<name of player/monster>,<initiative bonus (aka dex modifier)>
```
## TODO
If the player file changes between encounters (say due to level up or other penalty), the server needs to be restarted.
This can be annoying if a encounter starts when some players are surprised. 
