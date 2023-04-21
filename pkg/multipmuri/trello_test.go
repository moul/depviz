package multipmuri

import "fmt"

func ExampleNewTrelloCard() {
	entity := NewTrelloCard("uworIKRP")
	fmt.Println("entity")
	fmt.Println(" ", entity.String())
	fmt.Println(" ", entity.Kind())
	fmt.Println(" ", entity.Provider())

	relatives := []string{
		"https://trello.com/manfredtouron/boards",
		"https://trello.com/bertytech/home",
		"https://trello.com/manfredtouron",
		"https://trello.com/b/QO8ORojV/test",
		"https://trello.com/b/QO8ORojV",
		"https://trello.com/c/uworIKRP/1-card-1",
		"https://trello.com/c/uworIKRP",
		"https://trello.com/c/pqon4iMh/2-card-2#comment-5d79f330ae4a3225dff6faf9",
		"https://trello.com/c/pqon4iMh/blah#action-5d79f2b6053f4c137c9b9a28",
		"@manfredtouron",
	}
	fmt.Println("relationships")
	for _, name := range relatives {
		attrs := ""
		rel, err := entity.RelDecodeString(name)
		if err != nil {
			fmt.Printf("  %-80s -> error: %v\n", name, err)
			continue
		}
		if rel.Equals(entity) {
			attrs += " (equals)"
		}
		if entity.Contains(rel) {
			attrs += " (contains)"
		}
		if rel.Contains(entity) {
			attrs += " (is contained)"
		}
		fmt.Printf("  %-80s -> %s %s%s\n", name, rel.String(), rel.Kind(), attrs)
	}
	// Output:
	// entity
	//   https://trello.com/c/uworIKRP
	//   issue
	//   trello
	// relationships
	//   https://trello.com/manfredtouron/boards                                          -> https://trello.com/manfredtouron user-or-organization
	//   https://trello.com/bertytech/home                                                -> https://trello.com/bertytech user-or-organization
	//   https://trello.com/manfredtouron                                                 -> https://trello.com/manfredtouron user-or-organization
	//   https://trello.com/b/QO8ORojV/test                                               -> https://trello.com/b/QO8ORojV project
	//   https://trello.com/b/QO8ORojV                                                    -> https://trello.com/b/QO8ORojV project
	//   https://trello.com/c/uworIKRP/1-card-1                                           -> https://trello.com/c/uworIKRP issue (equals)
	//   https://trello.com/c/uworIKRP                                                    -> https://trello.com/c/uworIKRP issue (equals)
	//   https://trello.com/c/pqon4iMh/2-card-2#comment-5d79f330ae4a3225dff6faf9          -> https://trello.com/c/pqon4iMh issue
	//   https://trello.com/c/pqon4iMh/blah#action-5d79f2b6053f4c137c9b9a28               -> https://trello.com/c/pqon4iMh issue
	//   @manfredtouron                                                                   -> https://trello.com/manfredtouron user-or-organization

}
