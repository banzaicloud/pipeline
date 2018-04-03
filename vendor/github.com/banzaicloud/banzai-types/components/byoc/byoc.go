package byoc

type CreateBYOC struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (byoc *CreateBYOC) Validate() error {
	return nil
}
