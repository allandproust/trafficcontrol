package profileparameter

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
)

const (
	ProfileIDQueryParam   = "profileId"
	ParameterIDQueryParam = "parameterId"
)

//we need a type alias to define functions on
type TOProfileParameter struct {
	api.APIInfoImpl `json:"-"`
	tc.ProfileParameterNullable
}

// AllowMultipleCreates indicates whether an array can be POSTed using the shared Create handler
func (v *TOProfileParameter) AllowMultipleCreates() bool { return true }
func (v *TOProfileParameter) NewReadObj() interface{}    { return &tc.ProfileParametersNullable{} }
func (v *TOProfileParameter) SelectQuery() string        { return selectQuery() }
func (v *TOProfileParameter) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"profileId":   dbhelpers.WhereColumnInfo{"pp.profile", nil},
		"parameterId": dbhelpers.WhereColumnInfo{"pp.parameter", nil},
		"lastUpdated": dbhelpers.WhereColumnInfo{"pp.last_updated", nil},
	}
}
func (v *TOProfileParameter) DeleteQuery() string { return deleteQuery() }

func (pp TOProfileParameter) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{ProfileIDQueryParam, api.GetIntKey}, {ParameterIDQueryParam, api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (pp TOProfileParameter) GetKeys() (map[string]interface{}, bool) {
	if pp.ProfileID == nil {
		return map[string]interface{}{ProfileIDQueryParam: 0}, false
	}
	if pp.ParameterID == nil {
		return map[string]interface{}{ParameterIDQueryParam: 0}, false
	}
	keys := make(map[string]interface{})
	profileID := *pp.ProfileID
	parameterID := *pp.ParameterID

	keys[ProfileIDQueryParam] = profileID
	keys[ParameterIDQueryParam] = parameterID
	return keys, true
}

func (pp *TOProfileParameter) GetAuditName() string {
	if pp.ProfileID != nil {
		return strconv.Itoa(*pp.ProfileID) + "-" + strconv.Itoa(*pp.ParameterID)
	}
	return "unknown"
}

func (pp *TOProfileParameter) GetType() string {
	return "profileParameter"
}

func (pp *TOProfileParameter) SetKeys(keys map[string]interface{}) {
	profId, _ := keys[ProfileIDQueryParam].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	pp.ProfileID = &profId

	paramId, _ := keys[ParameterIDQueryParam].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	pp.ParameterID = &paramId
}

// Validate fulfills the api.Validator interface
func (pp *TOProfileParameter) Validate() error {

	errs := validation.Errors{
		"profile":   validation.Validate(pp.ProfileID, validation.Required),
		"parameter": validation.Validate(pp.ParameterID, validation.Required),
	}

	return util.JoinErrs(tovalidate.ToErrors(errs))
}

//The TOProfileParameter implementation of the Creator interface
//all implementations of Creator should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if a profileparameter with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
//The insert sql returns the profile and lastUpdated values of the newly inserted profileparameter and have
//to be added to the struct
func (pp *TOProfileParameter) Create() (error, error, int) {
	resultRows, err := pp.APIInfo().Tx.NamedQuery(insertQuery(), pp)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var profile int
	var parameter int
	var lastUpdated tc.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&profile, &parameter, &lastUpdated); err != nil {
			return nil, errors.New("profileparameter create scanning: " + err.Error()), http.StatusInternalServerError
		}
	}
	if rowsAffected == 0 {
		return nil, errors.New("profileparameter create returned no rows"), http.StatusInternalServerError
	}
	if rowsAffected > 1 {
		return nil, errors.New("profileparameter create returned multiple rows"), http.StatusInternalServerError
	}

	pp.SetKeys(map[string]interface{}{ProfileIDQueryParam: profile, ParameterIDQueryParam: parameter})
	return nil, nil, http.StatusOK
}

func insertQuery() string {
	return `INSERT INTO profile_parameter (
profile,
parameter) VALUES (
:profile_id,
:parameter_id) RETURNING profile, parameter, last_updated`
}

func (pp *TOProfileParameter) Update() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}
func (pp *TOProfileParameter) Read() ([]interface{}, error, error, int) { return api.GenericRead(pp) }
func (pp *TOProfileParameter) Delete() (error, error, int)              { return api.GenericDelete(pp) }

func selectQuery() string {

	query := `SELECT
pp.last_updated,
pp.parameter parameter_id,
prof.name profile
FROM profile_parameter pp
JOIN profile prof ON prof.id = pp.profile
JOIN parameter param ON param.id = pp.parameter`
	return query
}

func updateQuery() string {
	query := `UPDATE
profile_parameter SET
profile=:profile_id,
parameter=:parameter_id
WHERE profile=:profile_id AND
      parameter = :parameter_id
      RETURNING last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM profile_parameter
	WHERE profile=:profile_id and parameter=:parameter_id`
	return query
}
