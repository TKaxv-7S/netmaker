package logic

import (
	"encoding/json"
	"github.com/gravitl/netmaker/database"
	"github.com/gravitl/netmaker/models"
	"time"
)

// flags to keep for telemetry
var isFreeTier bool
var isEE bool

var lastSendTime int64 = 0

// setEEForTelemetry - store EE flag without having an import cycle when used for telemetry
// (as the ee package needs the logic package as currently written).
func SetEEForTelemetry(eeFlag bool) {
	isEE = eeFlag
}

// setFreeTierForTelemetry - store free tier flag without having an import cycle when used for telemetry
// (as the ee package needs the logic package as currently written).
func SetFreeTierForTelemetry(freeTierFlag bool) {
	isFreeTier = freeTierFlag
}

func sendTelemetry() error {
	return nil
}

func setTelemetryTimestamp(telRecord *models.Telemetry) error {
	lastSendTime = time.Now().Unix()
	return nil
}

// fetchTelemetryRecord - get the existing UUID and Timestamp from the DB
func fetchTelemetryRecord() (models.Telemetry, error) {
	var rawData string
	var telObj models.Telemetry
	var err error
	rawData, err = database.FetchRecord(database.SERVER_UUID_TABLE_NAME, database.SERVER_UUID_RECORD_KEY)
	if err != nil {
		return telObj, err
	}
	err = json.Unmarshal([]byte(rawData), &telObj)
	telObj.LastSend = lastSendTime
	return telObj, err
}
