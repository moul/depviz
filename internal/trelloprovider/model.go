package trelloprovider

import (
	"fmt"
	"github.com/cayleygraph/quad"
	"github.com/adlio/trello"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/depviz/v3/internal/dvparser"
)

func fromCards(cards []*trello.Card) dvmodel.Batch {
	batch := dvmodel.Batch{}
	for _, issue := range cards {
		err := fromCard(&batch, issue)
		if err != nil {
			fmt.Println("error")
			continue
		}
	}
	return batch
}

func fromCard(batch *dvmodel.Batch, input *trello.Card) error {
	
	entity, err := dvparser.ParseTarget(input.URL)
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}

	card := dvmodel.Task{
		ID:           quad.IRI(entity.String()),
		LocalID:      entity.LocalID(),
		Description:  input.Desc,
		Driver:       dvmodel.Driver_Trello,
		CreatedAt:    input.DateLastActivity,
		UpdatedAt:    input.DateLastActivity,
		Title:        input.Desc ,
		State:		  dvmodel.Task_Open,
		Kind: 		  dvmodel.Task_Card,
		HasOwner: 	  quad.IRI(entity.String()),
	}
	 
	owner := dvmodel.Owner{
		ID:          quad.IRI(entity.String()),
		LocalID:     entity.LocalID(),
		Kind:        dvmodel.Owner_User,
		// FullName:    "bob t",
		// ShortName:   "bob",
		Driver:      dvmodel.Driver_Trello,
		// Homepage:    "homepage",
		// AvatarURL:   "avatar-url",
		ForkStatus:  dvmodel.Owner_UnknownForkStatus,
		// Description: "description_owner",
	}

	topic := dvmodel.Topic{
		ID:          quad.IRI(entity.String()),
		LocalID:     entity.LocalID(),
		Title:       input.Name,
		Description: input.Desc,
		Kind: dvmodel.Topic_Label,
		Driver: dvmodel.Driver_Trello,
	}
	 
	batch.Tasks = append(batch.Tasks, &card)
	batch.Owners = append(batch.Owners, &owner)
	batch.Topics = append(batch.Topics, &topic)
	return nil
}