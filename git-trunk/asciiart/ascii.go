package asciiart

import (
	"github.com/salsita/SalsaFlow/git-trunk/log"
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
