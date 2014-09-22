package asciiart

import (
	"github.com/salsita/SalsaFlow/git-trunk/log"
)

func PrintThumbsUp() {
	// from http://emilights.com/ascii-auto-text-art/expression/217-thumbs-up
	log.Println("        __               __        ")
	log.Println("       (  |             |  )       ")
	log.Println("  _____ \\  \\           /  /_____   ")
	log.Println(" (____ _)   \\         /   (_____)  ")
	log.Println(" (_____ )  _)__(. .)__(_  ( _____) ")
	log.Println(" (__ ___)   )  |___|  (   (_  ___) ")
	log.Println("  (_____)__/   /_/\\_\\  \\__(____)   ")
}

func PrintScream(header, msg string) {
	log.Println(header)
	log.Println("     .----------.   ")
	log.Println("    /  .-.  .-.  \\  ")
	log.Println("   /   | |  | |   \\ ")
	log.Println("   \\   `-'  `-'  _/ ")
	log.Println("   /\\     .--.  / | ")
	log.Println("   \\ |   /  /  / /  ")
	log.Println("   / |  `--'  /\\ \\  ")
	log.Println("    /`-------'  \\ \\ ")
	log.Println(msg)
}
