package todos

type TodosItem struct {
	Id        int32  `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"Completed"`
}
