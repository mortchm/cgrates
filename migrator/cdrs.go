/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package migrator

import (
	"fmt"
	"time"

	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func (m *Migrator) migrateCurrentCDRs() (err error) {
	if m.sameStorDB { // no move
		return
	}
	cdrs, _, err := m.storDB.GetCDRs(new(utils.CDRsFilter), false)
	if err != nil {
		return err
	}
	/*
		for _, cdr := range cdrs {
			if err := m.oldStorDB.SetCDR(cdr, true); err != nil {
				return err
			}
		}
	*/
	return
}

func (m *Migrator) migrateCDRs() (err error) {
	var vrs engine.Versions
	vrs, err = m.dmIN.DataDB().GetVersions("")
	if err != nil {
		return utils.NewCGRError(utils.Migrator,
			utils.ServerErrorCaps,
			err.Error(),
			fmt.Sprintf("error: <%s> when querying oldDataDB for versions", err.Error()))
	} else if len(vrs) == 0 {
		return utils.NewCGRError(utils.Migrator,
			utils.MandatoryIEMissingCaps,
			utils.UndefinedVersion,
			"version number is not defined for Actions")
	}
	current := engine.CurrentDataDBVersions()
	switch vrs[utils.CDRs] {
	case current[utils.CDRs]:
		return m.migrateCurrentCDRs()
	}
	return
}

func (m *Migrator) migrateV1CDRs() (err error) {
	var v1CDR *v1Cdrs
	for {
		v1CDR, err = m.oldStorDB.getV1CDR()
		if err != nil && err != utils.ErrNoMoreData {
			return err
		}
		if err == utils.ErrNoMoreData {
			break
		}
		if v1CDR != nil {
			cdr := v1CDR.V1toV2Cdr()
			if m.dryRun != true {
				if err = m.storDB.SetCDR(cdr, true); err != nil {
					return err
				}
				m.stats[utils.CDRs] += 1
			}
		}
	}
	if m.dryRun != true {
		// All done, update version wtih current one
		vrs := engine.Versions{utils.CDRs: engine.CurrentStorDBVersions()[utils.CDRs]}
		if err = m.storDB.SetVersions(vrs, false); err != nil {
			return utils.NewCGRError(utils.Migrator,
				utils.ServerErrorCaps,
				err.Error(),
				fmt.Sprintf("error: <%s> when updating CDRs version into StorDB", err.Error()))
		}
	}
	return
}

//
type v1Cdrs struct {
	CGRID       string
	RunID       string
	OrderID     int64             // Stor order id used as export order id
	OriginHost  string            // represents the IP address of the host generating the CDR (automatically populated by the server)
	Source      string            // formally identifies the source of the CDR (free form field)
	OriginID    string            // represents the unique accounting id given by the telecom switch generating the CDR
	ToR         string            // type of record, meta-field, should map to one of the TORs hardcoded inside the server <*voice|*data|*sms|*generic>
	RequestType string            // matching the supported request types by the **CGRateS**, accepted values are hardcoded in the server <prepaid|postpaid|pseudoprepaid|rated>.
	Tenant      string            // tenant whom this record belongs
	Category    string            // free-form filter for this record, matching the category defined in rating profiles.
	Account     string            // account id (accounting subsystem) the record should be attached to
	Subject     string            // rating subject (rating subsystem) this record should be attached to
	Destination string            // destination to be charged
	SetupTime   time.Time         // set-up time of the event. Supported formats: datetime RFC3339 compatible, SQL datetime (eg: MySQL), unix timestamp.
	AnswerTime  time.Time         // answer time of the event. Supported formats: datetime RFC3339 compatible, SQL datetime (eg: MySQL), unix timestamp.
	Usage       time.Duration     // event usage information (eg: in case of tor=*voice this will represent the total duration of a call)
	ExtraFields map[string]string // Extra fields to be stored in CDR
	ExtraInfo   string            // Container for extra information related to this CDR, eg: populated with error reason in case of error on calculation
	Partial     bool              // Used for partial record processing by CDRC
	Rated       bool              // Mark the CDR as rated so we do not process it during rating
	CostSource  string            // The source of this cost
	Cost        float64
	CostDetails *engine.CallCost // Attach the cost details to CDR when possible
}

func (v1Cdr v1Cdrs) V1toV2Cdr() (cdr *engine.CDR) {
	cdr = &engine.CDR{
		CGRID:       v1Cdr.CGRID,
		RunID:       v1Cdr.RunID,
		OrderID:     v1Cdr.OrderID,
		OriginHost:  v1Cdr.OriginHost,
		Source:      v1Cdr.Source,
		OriginID:    v1Cdr.OriginID,
		ToR:         v1Cdr.ToR,
		RequestType: v1Cdr.RequestType,
		Tenant:      v1Cdr.Tenant,
		Category:    v1Cdr.Category,
		Account:     v1Cdr.Account,
		Subject:     v1Cdr.Subject,
		Destination: v1Cdr.Destination,
		SetupTime:   v1Cdr.SetupTime,
		AnswerTime:  v1Cdr.AnswerTime,
		Usage:       v1Cdr.Usage,
		ExtraFields: make(map[string]string),
		ExtraInfo:   v1Cdr.ExtraInfo,
		Partial:     v1Cdr.Partial,
		Rated:       v1Cdr.Rated,
		CostSource:  v1Cdr.CostSource,
		Cost:        v1Cdr.Cost,
		CostDetails: engine.NewEventCostFromCallCost(v1Cdr.CostDetails, v1Cdr.CGRID, v1Cdr.RunID),
	}
	if v1Cdr.ExtraFields != nil {
		for key, value := range v1Cdr.ExtraFields {
			cdr.ExtraFields[key] = value
		}
	}
	return
}