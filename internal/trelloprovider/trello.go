package trelloprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"github.com/adlio/trello"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/multipmuri"
	"go.uber.org/zap"
)

type Card struct {
	ShortLink string
}

type Opts struct {
	Since  *time.Time  `json:"since"`
	Logger *zap.Logger `json:"-"`
}

func FetchCard(ctx context.Context, entity multipmuri.Entity, token string, apikey string, boardid string, out chan<- dvmodel.Batch, opts Opts) {
	client := trello.NewClient(apikey, token)
	board, err := client.GetBoard(boardid)
	if err != nil {
	  fmt.Println()
	}
	cards, err := board.GetCards()
	if err != nil {
		opts.Logger.Error(err.Error(), zap.Error(err))
		return
	}
	batch := fromCards(cards, opts.Logger)
	out <- batch
}

func GetCardsId(BoardId string, token string, apikey string) ([]string, error) {
	url := `https://api.trello.com/1/boards/`+ BoardId + `/cards?key=` + apikey + `&token=` + token
	var cardsId []string
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
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

	var allCards []Card
	err = json.Unmarshal(resp, &allCards)
	if err != nil {
		return nil, err
	}
	
	var value string
	for i := 0; i < len(allCards); i++ {
		value = allCards[i].ShortLink
		if value == "" {
			break
		}
		cardsId = append(cardsId, value)
	}
	return cardsId, nil
}