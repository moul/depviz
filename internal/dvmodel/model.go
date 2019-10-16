package dvmodel

type Batch struct {
	Owners []Owner
	Tasks  []Task
	Topics []Topic
}

type Tasks []Task
