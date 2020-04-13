package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/justinian/dice"
)

type player struct {
	Name string
	Dex  int
}

type monster struct {
	Name string
	// HPRoll string
	Dex int
}

type Roll struct {
	Name string
	Roll int
}

type encounter struct {
	TurnOrder []Roll
	m         []monster
}

func (e *encounter) Hash(p []player) int64 {
	h := fnv.New64()
	for _, mo := range e.m {
		fmt.Fprintf(h, "%s%d", mo.Name, mo.Dex)
	}
	for _, p := range p {
		fmt.Fprintf(h, "%s%d", p.Name, p.Dex)
	}
	return int64(h.Sum64())
}

func (e *encounter) Roll(p []player) {
	var to []Roll
	rand.Seed(e.Hash(p))
	for _, pl := range p {
		r, _, err := dice.Roll(fmt.Sprintf("1d20%+d", pl.Dex))
		if err != nil {
			log.Fatal(err)
		}
		to = append(to, Roll{
			Name: pl.Name,
			Roll: r.Int(),
		})
	}
	for _, m := range e.m {
		r, _, err := dice.Roll(fmt.Sprintf("1d20%+d", m.Dex))
		if err != nil {
			log.Fatal(err)
		}
		to = append(to, Roll{
			Name: m.Name,
			Roll: r.Int(),
		})
	}
	sort.Slice(to, func(i, j int) bool {
		return to[i].Roll > to[j].Roll
	})
	e.TurnOrder = to
}

type server struct {
	Encounters []encounter
	p          []player
	pageTpl    *template.Template
}

func (s *server) EncounterAt(i int) encounter {
	if i < 0 || i > len(s.Encounters)-1 {
		return encounter{}
	}
	s.Encounters[i].Roll(s.p)
	return s.Encounters[i]
}

func (s *server) Play(w http.ResponseWriter, req *http.Request) {
	sp := strings.Split(req.URL.Path, "/")
	i, err := strconv.Atoi(sp[len(sp)-1])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err = s.pageTpl.Execute(w, s.EncounterAt(i))
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	return
}

var (
	bind string
	dir  string
)

func parseMonsterFile(f io.Reader) []monster {
	var m []monster
	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		dex, err := strconv.Atoi(strings.TrimSpace(rec[1]))
		if err != nil {
			log.Fatal(err)
		}
		m = append(m, monster{
			Name: rec[0],
			Dex:  dex,
		})
	}
	return m
}

func parsePlayerFile(f io.Reader) []player {
	var m []player
	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		dex, err := strconv.Atoi(strings.TrimSpace(rec[1]))
		if err != nil {
			log.Fatal(err)
		}
		m = append(m, player{
			Name: rec[0],
			Dex:  dex,
		})
	}
	return m
}

func getCSVFilesInDir(dir string) []string {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".csv" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func (s *server) LoadPlayers(dir string) {
	var pl []player
	for _, pfn := range getCSVFilesInDir(dir) {
		pf, err := os.Open(pfn)
		if err != nil {
			log.Fatal(err)
		}
		pl = append(pl, parsePlayerFile(pf)...)
	}
	s.p = pl
}

func (s *server) LoadMonsters(dir string) {
	var e []encounter
	for _, mfn := range getCSVFilesInDir(dir) {
		mf, err := os.Open(mfn)
		if err != nil {
			log.Fatal(err)
		}
		m := parseMonsterFile(mf)

		e = append(e, encounter{
			m: m,
		})
	}
	s.Encounters = e
}

func main() {
	flag.StringVar(&dir, "d", "./test", "")
	flag.StringVar(&bind, "b", "localhost:8080", "")
	flag.Parse()

	tpl, err := template.New("dm").Parse(dmHTML)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		pageTpl: tpl,
	}
	s.LoadPlayers(dir + "/players")
	s.watchPlayer(dir + "/players")

	s.LoadMonsters(dir + "/monsters")
	s.watchMonster(dir + "/monsters")

	http.HandleFunc("/encounter/", s.Play)
	log.Fatal(http.ListenAndServe(bind, nil))
}

func (s *server) watchPlayer(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("reloading players " + dir)
					s.LoadPlayers(dir)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) watchMonster(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("reloading monsters " + dir)
					s.LoadMonsters(dir)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}
}

var dmHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset='UTF-8'>
	<title>Inititatives of the 11 elephants</title>
	<style>
		.table {
			border-collapse: collapse;
			border-spacing: 0;
			empty-cells: show;
			border: 1px solid #cbcbcb;
		}
		.table td,
		.table th {
			border-left: 1px solid #cbcbcb;
			border-width: 0 0 0 1px;
			font-size: inherit;
			margin: 0;
			overflow: visible;
			padding: 0.5em 1em;
		}
		.table-bordered td {
			border-bottom: 1px solid #cbcbcb;
		}
		.active {
			background: yellow;
		}
	</style>
</head>
<body>
	<table id='initiative' class='table table-bordered'>
			<tr><th>Name</th><th>Initiative</th><th/></tr>{{range .TurnOrder}}
			<tr><td>{{.Name}}</td><td>{{.Roll}}</td><td><input type='button' value='kill' class='killable'/></td></tr>{{end}}
	</table>
	<input type='button' value='Next' class='next'/>
 	<script type='text/javascript'>
		var current = 1;
		document.querySelectorAll('.killable').forEach(function(i){
			i.addEventListener('click', function() {
				console.log(this);
				var i = this.parentNode.parentNode.rowIndex;
				document.getElementById('initiative').deleteRow(i);
				var tr = document.getElementById('initiative');
				update(tr);
			}, false);
		});
		function update(tr) {
			if (current > tr.rows.length-1) {
				current = 1;
			} 
			tr.rows[current].classList.add('active');
		}
		document.querySelector('.next').addEventListener('click', function() {
			var tr = document.getElementById('initiative');
			tr.rows[current].classList.remove('active');
			current++;
			update(tr);
		}, false);
		document.addEventListener('DOMContentLoaded', function() {
			var tr = document.getElementById('initiative');
			tr.rows[current].classList.add('active');			
		}, false);
	</script>
</body>
</html>
`
