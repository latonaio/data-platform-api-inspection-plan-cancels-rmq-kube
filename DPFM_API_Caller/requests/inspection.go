package requests

type Inspection struct {
	InspectionPlan int   `json:"InspectionPlan"`
	Inspection     int   `json:"Inspection"`
	IsCancelled    *bool `json:"IsCancelled"`
}
