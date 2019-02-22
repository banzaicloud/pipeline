// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package route53

import (
	"fmt"

	"github.com/banzaicloud/pipeline/config"
	route53model "github.com/banzaicloud/pipeline/dns/route53/model"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

// awsRoute53DatabaseStateStore is a database backed state store for
// managing the state of the domains registered by us in the Amazon Route53 external DNS service
type awsRoute53DatabaseStateStore struct{}

// create persists the given domain state to database
func (stateStore *awsRoute53DatabaseStateStore) create(state *domainState) error {
	db := config.DB()

	rec := createRoute53Domain(state)

	return db.Create(rec).Error
}

// update persists the changes of given domain state to database
func (stateStore *awsRoute53DatabaseStateStore) update(state *domainState) error {
	db := config.DB()

	dbRec := &route53model.Route53Domain{}
	err := db.Where(&route53model.Route53Domain{OrganizationId: state.organisationId, Domain: state.domain}).First(dbRec).Error
	if err != nil {
		return err
	}

	dbRec.Status = state.status
	dbRec.PolicyArn = state.policyArn
	dbRec.HostedZoneId = state.hostedZoneId
	dbRec.IamUser = state.iamUser
	dbRec.AwsAccessKeyId = state.awsAccessKeyId
	dbRec.ErrorMessage = state.errMsg

	return db.Save(dbRec).Error
}

// find looks up in the database the domain state identified by origId and domain. The found data is passed back
// through stateOut
func (stateStore *awsRoute53DatabaseStateStore) find(orgId pkgAuth.OrganizationID, domain string, stateOut *domainState) (bool, error) {
	db := config.DB()

	dbRec := &route53model.Route53Domain{}
	res := db.Where(&route53model.Route53Domain{OrganizationId: orgId, Domain: domain}).First(dbRec)

	if res.RecordNotFound() {
		return false, nil
	}
	err := res.Error
	if err != nil {
		return false, nil
	}

	initStateFromRoute53Domain(dbRec, stateOut)

	return true, nil
}

// listUnused returns all the domain state entries from database that belong to organizations with no live clusters
// thus the DNS domain entries earlier created for these domain are not used any more
func (stateStore *awsRoute53DatabaseStateStore) listUnused() ([]domainState, error) {
	db := config.DB()
	var dbRecs []route53model.Route53Domain

	sqlFilter := fmt.Sprintf("organization_id NOT IN (SELECT organization_id FROM clusters WHERE deleted_at is NULL AND status<>'%s')", cluster.Error)

	err := db.Where(&route53model.Route53Domain{Status: CREATED}).Where(sqlFilter).Find(&dbRecs).Error
	if err != nil {
		return nil, err
	}

	var domainStates []domainState

	for i := 0; i < len(dbRecs); i++ {
		var state domainState

		initStateFromRoute53Domain(&dbRecs[i], &state)
		domainStates = append(domainStates, state)
	}

	return domainStates, nil
}

// delete deletes domain state from database
func (stateStore *awsRoute53DatabaseStateStore) delete(state *domainState) error {
	db := config.DB()

	crit := &route53model.Route53Domain{OrganizationId: state.organisationId, Domain: state.domain}

	return db.Where(crit).Delete(&route53model.Route53Domain{}).Error
}

// findByStatus returns all the domain state entries from database that are in the specified status
func (stateStore *awsRoute53DatabaseStateStore) findByStatus(status string) ([]domainState, error) {
	db := config.DB()
	var dbRecs []route53model.Route53Domain

	crit := &route53model.Route53Domain{Status: status}
	err := db.Where(crit).Find(&dbRecs).Error
	if err != nil {
		return nil, err
	}

	var domainStates []domainState
	for i := 0; i < len(dbRecs); i++ {
		var state domainState

		initStateFromRoute53Domain(&dbRecs[i], &state)
		domainStates = append(domainStates, state)
	}

	return domainStates, nil
}

// findByOrgId looks up in the database the domain state identified by origId. The found data is passed back
// through stateOut
func (stateStore *awsRoute53DatabaseStateStore) findByOrgId(orgId pkgAuth.OrganizationID, stateOut *domainState) (bool, error) {
	db := config.DB()
	dbRec := &route53model.Route53Domain{}

	crit := &route53model.Route53Domain{OrganizationId: orgId}
	res := db.Where(crit).First(dbRec)

	if res.RecordNotFound() {
		return false, nil
	}
	err := res.Error
	if err != nil {
		return false, nil
	}

	initStateFromRoute53Domain(dbRec, stateOut)

	return true, nil
}

// createRoute53Domain create a new Route53Domain instance initialized from the passed in state
func createRoute53Domain(state *domainState) *route53model.Route53Domain {
	return &route53model.Route53Domain{
		OrganizationId: state.organisationId,
		Domain:         state.domain,
		Status:         state.status,
		PolicyArn:      state.policyArn,
		HostedZoneId:   state.hostedZoneId,
		IamUser:        state.iamUser,
		AwsAccessKeyId: state.awsAccessKeyId,
		ErrorMessage:   state.errMsg,
	}
}

// initStateFromRoute53Domain initializes state from the passed db record
func initStateFromRoute53Domain(dbRecord *route53model.Route53Domain, state *domainState) {
	state.createdAt = dbRecord.CreatedAt
	state.organisationId = dbRecord.OrganizationId
	state.domain = dbRecord.Domain
	state.status = dbRecord.Status
	state.policyArn = dbRecord.PolicyArn
	state.hostedZoneId = dbRecord.HostedZoneId
	state.iamUser = dbRecord.IamUser
	state.awsAccessKeyId = dbRecord.AwsAccessKeyId
	state.errMsg = dbRecord.ErrorMessage
}
