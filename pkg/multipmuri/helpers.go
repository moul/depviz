package multipmuri

func RepoEntity(source Entity) Entity {
	type hasRepoEntity interface {
		RepoEntity() Entity
	}
	if typed, found := source.(hasRepoEntity); found {
		return typed.RepoEntity()
	}
	return nil
}

func OwnerEntity(source Entity) Entity {
	type hasOwnerEntity interface {
		OwnerEntity() Entity
	}
	if typed, found := source.(hasOwnerEntity); found {
		return typed.OwnerEntity()
	}
	return nil
}

func ServiceEntity(source Entity) Entity {
	type hasServiceEntity interface {
		ServiceEntity() Entity
	}
	if typed, found := source.(hasServiceEntity); found {
		return typed.ServiceEntity()
	}
	return nil
}
