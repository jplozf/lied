package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

// Liste des mots de base du Lorem Ipsum
var loremWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "do", "eiusmod",
	"tempor", "incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua", "ut", "enim", "ad", "minim",
	"veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris", "nisi", "ut", "aliquip", "ex", "ea",
	"commodo", "consequat", "duis", "aute", "irure", "dolor", "in", "reprehenderit", "in", "voluptate",
	"velit", "esse", "cillum", "dolore", "eu", "fugiat", "nulla", "pariatur", "excepteur", "sint",
	"occaecat", "cupidatat", "non", "proident", "sunt", "in", "culpa", "qui", "officia", "deserunt",
	"mollit", "anim", "id", "est", "laborum",
}

// source initialisée une seule fois pour une meilleure performance
var random *rand.Rand

func init() {
	// Initialise le générateur de nombres aléatoires avec une graine (seed) unique
	// pour garantir des résultats différents à chaque exécution.
	source := rand.NewSource(time.Now().UnixNano())
	random = rand.New(source)
}

// capitalize met en majuscule la première lettre d'une chaîne de caractères.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// GenerateLoremIpsum génère un texte Lorem Ipsum aléatoire.
//
// Arguments :
//   - numParagraphs : le nombre de paragraphes à générer.
//   - minSentencesPerParagraph : le nombre minimum de phrases par paragraphe.
//   - maxSentencesPerParagraph : le nombre maximum de phrases par paragraphe.
//   - minWordsPerSentence : le nombre minimum de mots par phrase.
//   - maxWordsPerSentence : le nombre maximum de mots par phrase.
//
// Retourne une chaîne de caractères contenant le texte généré.
func GenerateLoremIpsum(numParagraphs, minSentencesPerParagraph, maxSentencesPerParagraph, minWordsPerSentence, maxWordsPerSentence int) string {
	// strings.Builder est plus efficace pour construire des chaînes de caractères en plusieurs fois.
	var sb strings.Builder

	for p := 0; p < numParagraphs; p++ {
		// Détermine le nombre de phrases pour ce paragraphe
		numSentences := random.Intn(maxSentencesPerParagraph-minSentencesPerParagraph+1) + minSentencesPerParagraph

		for s := 0; s < numSentences; s++ {
			// Détermine le nombre de mots pour cette phrase
			numWords := random.Intn(maxWordsPerSentence-minWordsPerSentence+1) + minWordsPerSentence

			var sentence strings.Builder
			for w := 0; w < numWords; w++ {
				// Choisit un mot au hasard dans notre liste
				word := loremWords[random.Intn(len(loremWords))]

				// Met une majuscule au premier mot de la phrase
				if w == 0 {
					word = capitalize(word)
				}

				sentence.WriteString(word)
				// Ajoute un espace après chaque mot (sauf le dernier)
				if w < numWords-1 {
					sentence.WriteString(" ")
				}
			}
			// Ajoute la phrase construite et un point final au paragraphe.
			sb.WriteString(sentence.String())
			sb.WriteString(".\n")
		}

		// Ajoute un saut de ligne entre les paragraphes (sauf pour le dernier)
		if p < numParagraphs-1 {
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

func main() {
	// --- Exemple d'utilisation ---
	fmt.Println("--- Paragraphe unique ---")
	text1 := GenerateLoremIpsum(1, 3, 5, 8, 15) // 1 paragraphe, de 3 à 5 phrases, de 8 à 15 mots par phrase
	fmt.Println(text1)

	fmt.Println("\n\n--- Plusieurs paragraphes ---")
	text2 := GenerateLoremIpsum(3, 4, 7, 5, 12) // 3 paragraphes, de 4 à 7 phrases, de 5 à 12 mots par phrase
	fmt.Println(text2)
}
