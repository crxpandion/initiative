package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	p         []player
	m         []monster
}

func (e *encounter) Roll() {
	for _, pl := range e.p {
		r, _, err := dice.Roll(fmt.Sprintf("1d20%+d", pl.Dex))
		if err != nil {
			log.Fatal(err)
		}
		e.TurnOrder = append(e.TurnOrder, Roll{
			Name: pl.Name,
			Roll: r.Int(),
		})
	}
	for _, m := range e.m {
		r, _, err := dice.Roll(fmt.Sprintf("1d20%+d", m.Dex))
		if err != nil {
			log.Fatal(err)
		}
		e.TurnOrder = append(e.TurnOrder, Roll{
			Name: m.Name,
			Roll: r.Int(),
		})
	}
	sort.Slice(e.TurnOrder, func(i, j int) bool {
		return e.TurnOrder[i].Roll > e.TurnOrder[j].Roll
	})
}

type server struct {
	Encounters []encounter
	pageTpl    *template.Template
}

func (s *server) EncounterAt(i int) encounter {
	if i < 0 || i > len(s.Encounters)-1 {
		return encounter{}
	}
	return s.Encounters[i]
}

func (s *server) Play(w http.ResponseWriter, req *http.Request) {
	i, err := strconv.Atoi(req.URL.Path[1:])
	log.Println(req.URL.Path)
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
	monstersFile string
	playersFile  string
	bind         string
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

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.StringVar(&monstersFile, "m", "./test/monsters.csv,./test/monsters_2.csv", "")
	flag.StringVar(&playersFile, "p", "./test/players.csv", "")
	flag.StringVar(&bind, "b", "localhost:8080", "")

	pf, err := os.Open(playersFile)
	if err != nil {
		log.Fatal(err)
	}
	players := parsePlayerFile(pf)

	e := make([]encounter, 0)
	for _, mf := range strings.Split(monstersFile, ",") {
		f, err := os.Open(mf)
		if err != nil {
			log.Fatal(err)
		}
		m := parseMonsterFile(f)

		enc := encounter{
			p: players,
			m: m,
		}
		enc.Roll()
		e = append(e, enc)
	}
	tpl, err := template.New("dm").Parse(dmHTML)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		Encounters: e,
		pageTpl:    tpl,
	}

	http.HandleFunc("/", s.Play)
	log.Fatal(http.ListenAndServe(bind, nil))
}

var dmHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset='UTF-8'>
	<title>Inititatives of the 11 elephants</title>
	<style>
	.pure-table {
		/* Remove spacing between table cells (from Normalize.css) */
		border-collapse: collapse;
		border-spacing: 0;
		empty-cells: show;
		border: 1px solid #cbcbcb;
	}
	
	.pure-table caption {
		color: #000;
		font: italic 85%/1 arial, sans-serif;
		padding: 1em 0;
		text-align: center;
	}
	
	.pure-table td,
	.pure-table th {
		border-left: 1px solid #cbcbcb;/*  inner column border */
		border-width: 0 0 0 1px;
		font-size: inherit;
		margin: 0;
		overflow: visible; /*to make ths where the title is really long work*/
		padding: 0.5em 1em; /* cell padding */
	}
	
	.pure-table thead {
		background-color: #e0e0e0;
		color: #000;
		text-align: left;
		vertical-align: bottom;
	}
	
	/* BORDERED TABLES */
	.pure-table-bordered td {
		border-bottom: 1px solid #cbcbcb;
	}
	.pure-table-bordered tbody > tr:last-child > td {
		border-bottom-width: 0;
	}
	.active {
		background: yellow;
	}
	</style>
</head>
<body>
	<table id='initiative' class='pure-table pure-table-bordered'>
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
