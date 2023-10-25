package requests

type Header struct {
	InspectionPlan int   `json:"InspectionPlan"`
	IsCancelled    *bool `json:"IsCancelled"`
}
