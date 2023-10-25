package dpfm_api_caller

import (
	"context"
	dpfm_api_input_reader "data-platform-api-inspection-plan-cancels-rmq-kube/DPFM_API_Input_Reader"
	dpfm_api_output_formatter "data-platform-api-inspection-plan-cancels-rmq-kube/DPFM_API_Output_Formatter"
	"data-platform-api-inspection-plan-cancels-rmq-kube/config"

	"github.com/latonaio/golang-logging-library-for-data-platform/logger"
	database "github.com/latonaio/golang-mysql-network-connector"
	rabbitmq "github.com/latonaio/rabbitmq-golang-client-for-data-platform"
	"golang.org/x/xerrors"
)

type DPFMAPICaller struct {
	ctx  context.Context
	conf *config.Conf
	rmq  *rabbitmq.RabbitmqClient
	db   *database.Mysql
}

func NewDPFMAPICaller(
	conf *config.Conf, rmq *rabbitmq.RabbitmqClient, db *database.Mysql,
) *DPFMAPICaller {
	return &DPFMAPICaller{
		ctx:  context.Background(),
		conf: conf,
		rmq:  rmq,
		db:   db,
	}
}

func (c *DPFMAPICaller) AsyncCancels(
	accepter []string,
	input *dpfm_api_input_reader.SDC,
	output *dpfm_api_output_formatter.SDC,
	log *logger.Logger,
) (interface{}, []error) {
	var response interface{}
	switch input.APIType {
	case "cancels":
		response = c.cancelSqlProcess(input, output, accepter, log)
	default:
		log.Error("unknown api type %s", input.APIType)
	}
	return response, nil
}

func (c *DPFMAPICaller) cancelSqlProcess(
	input *dpfm_api_input_reader.SDC,
	output *dpfm_api_output_formatter.SDC,
	accepter []string,
	log *logger.Logger,
) *dpfm_api_output_formatter.Message {
	var headerData *dpfm_api_output_formatter.Header
	inspectionData := make([]dpfm_api_output_formatter.Inspection, 0)
	for _, a := range accepter {
		switch a {
		case "Header":
			h, i := c.headerCancel(input, output, log)
			headerData = h
			if h == nil || i == nil {
				continue
			}
			inspectionData = append(inspectionData, *i...)
		case "Inspection":
			i := c.inspectionCancel(input, output, log)
			if i == nil {
				continue
			}
			inspectionData = append(inspectionData, *i...)
		}
	}

	return &dpfm_api_output_formatter.Message{
		Header:     headerData,
		Inspection: &inspectionData,
	}
}

func (c *DPFMAPICaller) headerCancel(
	input *dpfm_api_input_reader.SDC,
	output *dpfm_api_output_formatter.SDC,
	log *logger.Logger,
) (*dpfm_api_output_formatter.Header, *[]dpfm_api_output_formatter.Inspection) {
	sessionID := input.RuntimeSessionID

	header := c.HeaderRead(input, log)
	if header == nil {
		return nil, nil, nil, nil
	}
	header.IsCancelled = input.Header.IsCancelled
	res, err := c.rmq.SessionKeepRequest(nil, c.conf.RMQ.QueueToSQL()[0], map[string]interface{}{"message": header, "function": "InspectionPlanHeader", "runtime_session_id": sessionID})
	if err != nil {
		err = xerrors.Errorf("rmq error: %w", err)
		log.Error("%+v", err)
		return nil, nil, nil, nil
	}
	res.Success()
	if !checkResult(res) {
		output.SQLUpdateResult = getBoolPtr(false)
		output.SQLUpdateError = "Header Data cannot cancel"
		return nil, nil, nil, nil
	}
	// headerのキャンセルが取り消された時は子に影響を与えない
	if !*header.IsCancelled {
		return header, nil, nil, nil
	}

	inspections := c.InspectionsRead(input, log)
	for i := range *inspections {
		(*inspections)[i].IsCancelled = input.Header.IsCancelled
		res, err := c.rmq.SessionKeepRequest(nil, c.conf.RMQ.QueueToSQL()[0], map[string]interface{}{"message": (*items)[i], "function": "InspectionPlanInspection", "runtime_session_id": sessionID})
		if err != nil {
			err = xerrors.Errorf("rmq error: %w", err)
			log.Error("%+v", err)
			return nil, nil, nil, nil
		}
		res.Success()
		if !checkResult(res) {
			output.SQLUpdateResult = getBoolPtr(false)
			output.SQLUpdateError = "Inspection Data cannot cancel"
			return nil, nil, nil, nil
		}
	}

	return header, inspections
}

func (c *DPFMAPICaller) inspectionCancel(
	input *dpfm_api_input_reader.SDC,
	output *dpfm_api_output_formatter.SDC,
	log *logger.Logger,
) *[]dpfm_api_output_formatter.Inspection {
	sessionID := input.RuntimeSessionID

	inspections := make([]dpfm_api_output_formatter.Inspection, 0)
	for _, v := range input.Header.Inspection {
		data := dpfm_api_output_formatter.Inspection{
			InspectionPlan: input.Header.InspectionPlan,
			Inspection:     v.Inspection,
			IsCancelled:    v.IsCancelled,
		}
		res, err := c.rmq.SessionKeepRequest(nil, c.conf.RMQ.QueueToSQL()[0], map[string]interface{}{
			"message":            data,
			"function":           "InspectionPlanInspection",
			"runtime_session_id": sessionID,
		})
		if err != nil {
			err = xerrors.Errorf("rmq error: %w", err)
			log.Error("%+v", err)
			return nil
		}
		res.Success()
		if !checkResult(res) {
			output.SQLUpdateResult = getBoolPtr(false)
			output.SQLUpdateError = "Inspection Data cannot cancel"
			return nil
		}
	}
	// inspectionがキャンセル取り消しされた場合、headerのキャンセルも取り消す
	if !*input.Header.Inspection[0].IsCancelled {
		header := c.HeaderRead(input, log)
		header.IsCancelled = input.Header.Inspection[0].IsCancelled
		res, err := c.rmq.SessionKeepRequest(nil, c.conf.RMQ.QueueToSQL()[0], map[string]interface{}{"message": header, "function": "InspectionPlanHeader", "runtime_session_id": sessionID})
		if err != nil {
			err = xerrors.Errorf("rmq error: %w", err)
			log.Error("%+v", err)
			return nil
		}
		res.Success()
		if !checkResult(res) {
			output.SQLUpdateResult = getBoolPtr(false)
			output.SQLUpdateError = "Header Data cannot cancel"
			return nil
		}
	}

	return &items
}

func checkResult(msg rabbitmq.RabbitmqMessage) bool {
	data := msg.Data()
	d, ok := data["result"]
	if !ok {
		return false
	}
	result, ok := d.(string)
	if !ok {
		return false
	}
	return result == "success"
}

func getBoolPtr(b bool) *bool {
	return &b
}
