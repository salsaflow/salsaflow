package asciiart

import (
	"github.com/salsita/salsaflow/log"
)

func PrintThumbsUp() {
	// from http://emilights.com/ascii-auto-text-art/expression/217-thumbs-up
	log.Println(`        __               __
       (  |             |  )
  _____ \  \           /  /_____
 (____ _)   \   ___   /   (_____)
 (_____ )  _)__(. .)__(_  ( _____)
 (__ ___)   )  |___|  (   (_  ___)
  (_____)__/   /_/\_\  \__(____)`)
}

func PrintScream(header, msg string) {
	log.Println(header)
	log.Println(`   .----------. 
  /  .-.  .-.  \
 /   | |  | |   \
 \   ` + "`" + `-'  ` + "`" + `-'  _/
 /\     .--.  / |
 \ |   /  /  / /
 / |  ` + "`" + `--'  /\ \
  /` + "`" + `-------'  \ \`)
	log.Println(msg)
}

func PrintSnoopy() {
	log.Println(`
    ,-~~-.___.
   / |  '     \
  (  )         0    Let's do this!
   \_/-, ,----'
      ====           //
     /  \-'~;    /~~~(O)
    /  __/~|   /       |
  =(  _____| (_________|
`)
}

func PrintGrimReaper(msg string) {
	log.Printf(`
                ( %v )
    ___o .--.  o
   /___| |OO| .
  /'   |_|  |
       (_    _)
       | |   \
       | |oo_/

`, msg)
}
