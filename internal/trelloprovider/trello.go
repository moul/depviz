package trelloprovider

import (
	"context"
	"fmt"
	"github.com/adlio/trello"
	"moul.io/multipmuri"
	"moul.io/depviz/v3/internal/dvmodel"
	"net/http"
	"io/ioutil"
	"strconv"
	"github.com/tidwall/gjson"
)

func FetchCard(ctx context.Context, entity multipmuri.Entity, token string, apikey string, boardid string, out chan<- dvmodel.Batch) {
	client := trello.NewClient(apikey, token)
	board, err := client.GetBoard(boardid)
	if err != nil {
	  fmt.Println()
	}
	cards, _ := board.GetCards()
	batch := fromCards(cards)
	out <- batch
}

func GetCardsId(BoardId string, token string, apikey string) ([]string, error) {
	url := `https://api.trello.com/1/boards/`+ BoardId + `/cards?key=` + apikey + `&token=` + token
	var cardsId []string
	req, err := http.NewRequest("GET", url, nil)

	if (err != nil) {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	BoardGetResponse, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer BoardGetResponse.Body.Close()
	resp, err := ioutil.ReadAll(BoardGetResponse.Body)

	if err != nil {
		return nil, err
	}
	i := 0
	var value gjson.Result
	for {
		value = gjson.Get(string(resp), strconv.Itoa(i) + `.shortLink`)
		if value.String() == "" {
			break
		}
		cardsId = append(cardsId, value.String())
		i++
	}
	return cardsId, nil
}