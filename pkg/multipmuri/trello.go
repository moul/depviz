package multipmuri

import (
	"fmt"
	"net/url"
	"strings"
)

//
// TrelloService
//

type TrelloService struct {
	Service
}

func NewTrelloService() *TrelloService {
	return &TrelloService{
		Service: &service{},
	}
}

func (e TrelloService) Provider() Provider {
	return TrelloProvider
}

func (e TrelloService) String() string {
	return "https://trello.com/"
}

func (e TrelloService) LocalID() string {
	return "trello.com"
}

func (e TrelloService) RelDecodeString(input string) (Entity, error) {
	return trelloRelDecodeString(input, false)
}

func (e TrelloService) Equals(other Entity) bool {
	_, valid := other.(*TrelloService)
	return valid
}

func (e TrelloService) Contains(other Entity) bool {
	return true
}

//
// TrelloCard
//

type TrelloCard struct {
	Issue
	*withTrelloID
}

func NewTrelloCard(id string) *TrelloCard {
	return &TrelloCard{
		Issue:        &issue{},
		withTrelloID: &withTrelloID{id},
	}
}

func (e TrelloCard) String() string {
	return fmt.Sprintf("https://trello.com/c/%s", e.ID())
}

func (e TrelloCard) LocalID() string {
	return fmt.Sprintf("c/%s", e.ID())
}

func (e TrelloCard) RelDecodeString(input string) (Entity, error) {
	return trelloRelDecodeString(input, false)
}

func (e TrelloCard) Equals(other Entity) bool {
	if typed, valid := other.(*TrelloCard); valid {
		return e.ID() == typed.ID()
	}
	return false
}

func (e TrelloCard) Contains(other Entity) bool {
	return false
}

//
// TrelloUser
//

type TrelloUser struct {
	UserOrOrganization
	*withTrelloID
}

func NewTrelloUser(id string) *TrelloUser {
	return &TrelloUser{
		UserOrOrganization: &userOrOrganization{},
		withTrelloID:       &withTrelloID{id},
	}
}

func (e TrelloUser) String() string {
	return fmt.Sprintf("https://trello.com/%s", e.ID())
}

func (e TrelloUser) LocalID() string {
	return fmt.Sprintf("@%s", e.ID())
}

func (e TrelloUser) RelDecodeString(input string) (Entity, error) {
	return trelloRelDecodeString(input, false)
}

func (e TrelloUser) Equals(other Entity) bool {
	if typed, valid := other.(*TrelloUser); valid {
		return e.ID() == typed.ID()
	}
	return false
}

func (e TrelloUser) Contains(other Entity) bool {
	return false
}

//
// TrelloBoard
//

type TrelloBoard struct {
	Project
	*withTrelloID
}

func NewTrelloBoard(id string) *TrelloBoard {
	return &TrelloBoard{
		Project:      &project{},
		withTrelloID: &withTrelloID{id},
	}
}

func (e TrelloBoard) String() string {
	return fmt.Sprintf("https://trello.com/b/%s", e.ID())
}

func (e TrelloBoard) LocalID() string {
	return fmt.Sprintf("b/%s", e.ID())
}
func (e TrelloBoard) RelDecodeString(input string) (Entity, error) {
	return trelloRelDecodeString(input, false)
}

func (e TrelloBoard) Equals(other Entity) bool {
	if typed, valid := other.(*TrelloBoard); valid {
		return e.ID() == typed.ID()
	}
	return false
}

func (e TrelloBoard) Contains(other Entity) bool {
	return false
}

//
// TrelloCommon
//

type withTrelloID struct{ id string }

func (e *withTrelloID) Provider() Provider { return TrelloProvider }
func (e *withTrelloID) ID() string         { return e.id }

//
// Helpers
//

func trelloRelDecodeString(input string, force bool) (Entity, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}
	if isProviderScheme(u.Scheme) { // trello://, trello://, ...
		return DecodeString(input)
	}
	u.Path = strings.Trim(u.Path, "/")
	if u.Host == "" && len(u.Path) > 0 { // domain.com/a/b
		u.Host = getHostname(u.Path)
		if u.Host != "" {
			u.Path = u.Path[len(u.Host):]
			u.Path = strings.Trim(u.Path, "/")
		}
	}
	//if owner != "" && repo != "" && strings.HasPrefix(u.Path, "@") { // @manfredtouron from a card
	//return NewTrelloUser(u.Path[1:]), nil
	//}
	if u.Path == "" && u.Fragment == "" {
		return NewTrelloService(), nil
	}
	parts := strings.Split(u.Path, "/")
	lenParts := len(parts)
	// FIXME: handle fragment (actions, comments, etc)
	//log.Println("path", u.Path, "fragment", u.Fragment, "len", lenParts)
	switch {
	case lenParts > 1 && parts[0] == "c":
		return NewTrelloCard(parts[1]), nil
	case lenParts > 1 && parts[0] == "b":
		return NewTrelloBoard(parts[1]), nil
	default:
		if parts[0][0] == '@' {
			return NewTrelloUser(parts[0][1:]), nil
		}
		return NewTrelloUser(parts[0]), nil
	}
}
