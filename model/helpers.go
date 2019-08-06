package model

func (i *Issue) PostLoad() {
	i.ParentIDs = []string{}
	i.ChildIDs = []string{}
	i.DuplicateIDs = []string{}
	for _, rel := range i.Parents {
		i.ParentIDs = append(i.ParentIDs, rel.ID)
	}
	for _, rel := range i.Children {
		i.ChildIDs = append(i.ChildIDs, rel.ID)
	}
	for _, rel := range i.Duplicates {
		i.DuplicateIDs = append(i.DuplicateIDs, rel.ID)
	}
}
